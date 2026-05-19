// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/MultiHopStep.sol";
import {IERC20} from "openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";
import {MarketParams} from "morpho-blue/src/interfaces/IMorpho.sol";

interface IOracle {
    function price() external view returns (uint256);
}

interface ISwapRouter {
    struct ExactInputSingleParams {
        address tokenIn;
        address tokenOut;
        uint24  fee;
        address recipient;
        uint256 amountIn;
        uint256 amountOutMinimum;
        uint160 sqrtPriceLimitX96;
    }
    function exactInputSingle(ExactInputSingleParams calldata params)
        external returns (uint256 amountOut);
}

contract LiquidatorMultiHopTest is Test {
    address constant MORPHO = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;

    // SwapRouter02 UniV3 — c'est lui qui exécute les swaps
    address constant ROUTER = 0x2626664c2603336E57B271c5C0b26F421741e481;

    // QuoterV2 UniV3 — NE PAS utiliser ici, c'est read-only
    // address constant QUOTER = 0x3d4e44eb1374240ce5f1b871ab261cd16335b76a;

    // Tokens Base
    address constant WETH  = 0x4200000000000000000000000000000000000006;
    address constant USDC  = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;
    address constant cbETH = 0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22;
    address constant cbBTC = 0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf;

    function setUp() public {
        vm.createSelectFork(vm.envString("BASE_RPC"));
    }

    // ─────────────────────────────────────────────────────────────────────────
    // 1-hop : WETH → USDC  (contrôle — reprend le test d'origine)
    // ─────────────────────────────────────────────────────────────────────────
    function test_1hop_WETH_USDC() public {
        Liquidator.SwapStep[] memory steps = new Liquidator.SwapStep[](1);
        steps[0] = _buildStep(WETH, USDC, 500);

        _liquidate({
            blockNum:        45433873,
            loanToken:       USDC,
            collateralToken: WETH,
            oracle:          0xD09048c8B568Dbf5f189302beA26c9edABFC4858,
            irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
            lltv:            860000000000000000,
            borrower:        0x924D53C12e04B7F74D97c9770889502D7aecc95e,
            seizedAssets:    8984643,
            steps:           steps
        });
    }

    // ─────────────────────────────────────────────────────────────────────────
    // 2-hop : cbETH → WETH → USDC
    //
    // Marché : loan=USDC, collateral=cbETH
    // cbETH→WETH (fee 500) puis WETH→USDC (fee 500)
    // ─────────────────────────────────────────────────────────────────────────
    function test_2hop_cbETH_WETH_USDC() public {
        Liquidator.SwapStep[] memory steps = new Liquidator.SwapStep[](2);
        steps[0] = _buildStep(cbETH, WETH, 500);
        steps[1] = _buildStep(WETH,  USDC, 500);

        _liquidate({
            blockNum:        45433873,
            loanToken:       USDC,
            collateralToken: cbETH,
            oracle:          0xD09048c8B568Dbf5f189302beA26c9edABFC4858,
            irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
            lltv:            860000000000000000,
            borrower:        0x924D53C12e04B7F74D97c9770889502D7aecc95e,
            seizedAssets:    1e18,
            steps:           steps
        });
    }

    // ─────────────────────────────────────────────────────────────────────────
    // 2-hop : cbBTC → WETH → USDC
    //
    // Marché : loan=USDC, collateral=cbBTC
    // cbBTC→WETH (fee 3000) puis WETH→USDC (fee 500)
    // ─────────────────────────────────────────────────────────────────────────
    function test_2hop_cbBTC_WETH_USDC() public {
        Liquidator.SwapStep[] memory steps = new Liquidator.SwapStep[](2);
        steps[0] = _buildStep(cbBTC, WETH, 3000);
        steps[1] = _buildStep(WETH,  USDC, 500);

        _liquidate({
            blockNum:        45433873,
            loanToken:       USDC,
            collateralToken: cbBTC,
            oracle:          0xD09048c8B568Dbf5f189302beA26c9edABFC4858,
            irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
            lltv:            860000000000000000,
            borrower:        0x924D53C12e04B7F74D97c9770889502D7aecc95e,
            seizedAssets:    142096,
            steps:           steps
        });
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Helpers
    // ─────────────────────────────────────────────────────────────────────────

    /// @notice Construit une SwapStep UniV3 SwapRouter02 exactInputSingle.
    ///
    /// Layout ABI après le sélecteur (4 bytes) :
    ///   +4    tokenIn          slot 0
    ///   +36   tokenOut         slot 1
    ///   +68   fee              slot 2
    ///   +100  recipient        slot 3  ← patché dans _liquidate
    ///   +132  amountIn         slot 4  ← amountInOffset = 132
    ///   +164  amountOutMinimum slot 5
    ///   +196  sqrtPriceLimitX96 slot 6
    function _buildStep(
        address tokenIn,
        address tokenOut,
        uint24  fee
    ) internal pure returns (Liquidator.SwapStep memory) {
        bytes memory data = abi.encodeCall(
            ISwapRouter.exactInputSingle,
            ISwapRouter.ExactInputSingleParams({
                tokenIn:           tokenIn,
                tokenOut:          tokenOut,
                fee:               fee,
                recipient:         address(0), // patché dans _liquidate
                amountIn:          0,          // patché au runtime par le contrat
                amountOutMinimum:  0,
                sqrtPriceLimitX96: 0
            })
        );

        return Liquidator.SwapStep({
            target:         ROUTER,
            data:           data,
            tokenIn:        tokenIn,
            tokenOut:       tokenOut,
            amountInOffset: 132  // 4 + 4*32
        });
    }

    function _liquidate(
        uint256                      blockNum,
        address                      loanToken,
        address                      collateralToken,
        address                      oracle,
        address                      irm,
        uint256                      lltv,
        address                      borrower,
        uint256                      seizedAssets,
        Liquidator.SwapStep[] memory steps
    ) internal {
        vm.rollFork(blockNum);

        Liquidator liquidator = new Liquidator(MORPHO);
        liquidator.setTarget(ROUTER, true);

        // Patch recipient (slot 3, offset 100) dans chaque step
        for (uint256 i = 0; i < steps.length; i++) {
            bytes memory d   = steps[i].data;
            address      liq = address(liquidator);
            assembly {
                mstore(add(add(d, 32), 100), liq)
            }
            steps[i].data = d;
        }

        MarketParams memory mp = MarketParams({
            loanToken:       loanToken,
            collateralToken: collateralToken,
            oracle:          oracle,
            irm:             irm,
            lltv:            lltv
        });

        uint256 price = IOracle(oracle).price();
        vm.mockCall(oracle, abi.encodeWithSignature("price()"), abi.encode(price * 90 / 100));

        liquidator.liquidate(mp, borrower, seizedAssets, 0, steps, 0);

        uint256 profit     = IERC20(loanToken).balanceOf(address(liquidator));
        uint256 profitColl = IERC20(collateralToken).balanceOf(address(liquidator));

        console.log("profit loanToken :", profit);
        console.log("profit collToken :", profitColl);
        assertGt(profit + profitColl, 0, "no profit");
    }
}
