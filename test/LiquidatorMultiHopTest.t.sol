// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/MultiHopStep.sol";
import {IERC20} from "openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";
import {MarketParams} from "morpho-blue/src/interfaces/IMorpho.sol";
import {Id, Position} from "morpho-blue/src/interfaces/IMorpho.sol";


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

struct Route {
        address from;
        address to;
        bool stable;
        address factory;
    }

        struct SwapExactTokensForTokensParams {
            uint256 amountIn;
            uint256 amountOutMin;
            Route[] routes;
            address to;
            uint256 deadline;
        }
    function swapExactTokensForTokens(
        uint256 amountIn,
        uint256 amountOutMin,
        Route[] calldata routes,
        address to,
        uint256 deadline
    ) external returns (uint256[] memory amounts);

   
  
}

contract LiquidatorMultiHopTest is Test {
    address constant MORPHO = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;

    // SwapRouter02 UniV3 — c'est lui qui exécute les swaps
    address constant UNIROUTER = 0x2626664c2603336E57B271c5C0b26F421741e481;
    address constant AEROROUTER = 0xcF77a3Ba9A5CA399B7c97c74d54e5b1Beb874E43;
    address constant SLIPSTREAM_ROUTER = 0xBE6D8f0d05cC4be24d5167a3eF062215bE6D18a5;
    address constant PANROUTER = 0x1B8513744261E0F3c962A728599347a0D9F96C8d;
    // QuoterV2 UniV3 — NE PAS utiliser ici, c'est read-only
    // address constant QUOTER = 0x3d4e44eb1374240ce5f1b871ab261cd16335b76a;

    // Tokens Base
    address constant WETH  = 0x4200000000000000000000000000000000000006;
    address constant USDC  = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913;
    address constant cbETH = 0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22;
    address constant weETH = 0x04C0599Ae5A44757c0af6F9eC3b93da8976c150A;
    address constant wstETH = 0x59b39314807216a498f478d06A2e15C72C396F1e;

    function setUp() public {
        vm.createSelectFork(vm.envString("BASE_RPC"));
    }

    // ─────────────────────────────────────────────────────────────────────────
    // 1-hop : WETH → USDC  (contrôle — reprend le test d'origine)
    // ─────────────────────────────────────────────────────────────────────────
    function test_1hop_WETH_USDC() public {
        Liquidator.SwapStep[] memory steps = new Liquidator.SwapStep[](1);
        steps[0] = _buildUniStep(WETH, USDC, 500);

        _liquidate({
            blockNum:        46170688,  // block n-1
            loanToken:       USDC,
            collateralToken: WETH,
            oracle:          0xFEa2D58cEfCb9fcb597723c6bAE66fFE4193aFE4,
            irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
            lltv:            860000000000000000,
            borrower:        0x33B8F8Ee093F64eEC64eF619C1C291D89b1fc4C4,
            seizedAssets:    32887891089030540,
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
        steps[0] = _buildUniStep(cbETH, WETH, 500);
        steps[1] = _buildUniStep(WETH,  USDC, 500);

        _liquidate({
            blockNum:        46066882,
            loanToken:       USDC,
            collateralToken: cbETH,
            oracle:          0xb40d93F44411D8C09aD17d7F88195eF9b05cCD96,
            irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
            lltv:            860000000000000000,
            borrower:        0x65A72471F4159B359f9fC5de946ed7Ac3b411618,
            seizedAssets:    1038472279785147941,
            steps:           steps
        });
    }

     function test_1hop_pankake_cbETH_USDC() public {
        // Call PancakeSwap's Quoter contract directly in your test

        Liquidator.SwapStep[] memory steps = new Liquidator.SwapStep[](1);
        steps[0] = _buildPanStep(cbETH,  USDC, 100);

        _liquidate({
            blockNum:        46170694,
            loanToken:       USDC,
            collateralToken: cbETH,
            oracle:          0x97FF9CbD7E77348b2B8FfBB883bF29452aD18295,
            irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
            lltv:            770000000000000000,
            borrower:        0x1674DE8a1c8208f8f1185cd80030B54Ee741DeEb,
            seizedAssets:    8881540695283522,
            steps:           steps
        });
    }






    // ─────────────────────────────────────────────────────────────────────────
    // 2-hop : cbBTC → WETH → USDC
    //
    // Marché : loan=USDC, collateral=cbBTC
    // cbBTC→WETH (fee 3000) puis WETH→USDC (fee 500)
    // ─────────────────────────────────────────────────────────────────────────
   

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

    function _buildUniStep(
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
            target:         UNIROUTER,
            data:           data,
            tokenIn:        tokenIn,
            tokenOut:       tokenOut,
            amountInOffset: 132  // 4 + 4*32
        });
    }


function _buildPanStep(
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
            target:         PANROUTER,
            data:           data,
            tokenIn:        tokenIn,
            tokenOut:       tokenOut,
            amountInOffset: 132  // 4 + 4*32
        });
    }
    // layout après sélecteur :
    // +4   tokenIn      slot0
    // +36  tokenOut     slot1
    // +68  tickSpacing  slot2
    // +100 recipient    slot3  ← patché dans _liquidate (offset 100)
    // +132 deadline     slot4
    // +164 amountIn     slot5  ← amountInOffset = 164

   


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
        
        uint256 price = IOracle(oracle).price();
        vm.mockCall(oracle, abi.encodeWithSignature("price()"), abi.encode(price * 70 / 100));


        Liquidator liquidator = new Liquidator(MORPHO);
        liquidator.setTarget(UNIROUTER, true);
    liquidator.setTarget(PANROUTER, true);
   
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

        Id mId = MarketParamsLib.id(mp);
        Position memory p = IMorpho(MORPHO).position(mId, borrower);
        console.log("collShares :", p.collateral);
        console.log("borrowShares:",p.borrowShares);
        console.logBytes(steps[0].data);
        liquidator.liquidate(mp, borrower, seizedAssets, 0, steps, 0);

        uint256 profit     = IERC20(loanToken).balanceOf(address(liquidator));
        uint256 profitColl = IERC20(collateralToken).balanceOf(address(liquidator));

        console.log("profit loanToken :", profit);
        console.log("profit collToken :", profitColl);
        assertGt(profit + profitColl, 0, "no profit");
    }
}
