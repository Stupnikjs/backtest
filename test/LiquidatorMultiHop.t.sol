// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/MultiHopStep.sol";
import {IERC20} from "openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";
import {MarketParams} from "morpho-blue/src/interfaces/IMorpho.sol";

interface IOracle {
    function price() external view returns (uint256);
}

// Interface minimale du router Uniswap V3
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

contract LiquidatorTest is Test {
    Liquidator.SwapStep step; 
    address constant MORPHO     = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;
    address constant UNI_ROUTER = 0x2626664c2603336E57B271c5C0b26F421741e481;

    function setUp() public {
        vm.createSelectFork(vm.envString("BASE_RPC"));
    }

    // ─────────────────────────────────────────────
    // Tests
    // ─────────────────────────────────────────────

    // Marché WETH/USDC — le collatéral est USDC, le loan est WETH
    // On reçoit du USDC (collat), on doit rembourser du WETH (loan)
    // => swap USDC → WETH
    function test_WETH_USDC() public {
        address loanToken       = 0x4200000000000000000000000000000000000006; // WETH
        address collateralToken = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913; // USDC

        Liquidator.SwapStep[] memory steps = _buildSingleHopSteps(
            collateralToken, // tokenIn  : ce qu'on reçoit du collatéral
            loanToken,       // tokenOut : ce qu'on doit rembourser
            500              // fee pool Uni V3
        );

        _liquidate({
            blockNum:        45433873,
            loanToken:       loanToken,
            collateralToken: collateralToken,
            oracle:          0xD09048c8B568Dbf5f189302beA26c9edABFC4858,
            irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
            lltv:            860000000000000000,
            borrower:        0x924D53C12e04B7F74D97c9770889502D7aecc95e,
            seizedAssets:    8984643,
            steps:           steps
        });
    }

    // Marché USDC/WETH — le collatéral est WETH, le loan est USDC
    // On reçoit du WETH (collat), on doit rembourser du USDC (loan)
    // => swap WETH → USDC
    function test_USDC_WETH() public {
        address loanToken       = 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913; // USDC
        address collateralToken = 0x4200000000000000000000000000000000000006; // WETH

        Liquidator.SwapStep[] memory steps = _buildSingleHopSteps(
            collateralToken, // tokenIn  : WETH reçu
            loanToken,       // tokenOut : USDC à rembourser
            500
        );

        _liquidate({
            blockNum:        45379997,
            loanToken:       loanToken,
            collateralToken: collateralToken,
            oracle:          0xFEa2D58cEfCb9fcb597723c6bAE66fFE4193aFE4,
            irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
            lltv:            860000000000000000,
            borrower:        0xb8Aa8a5E09245c6D4FAf602020853dE595d9f14A,
            seizedAssets:    4624239072469418,
            steps:           steps
        });
    }

    // ─────────────────────────────────────────────
    // Helpers
    // ─────────────────────────────────────────────

    /// @notice Construit un tableau d'une seule SwapStep pour un swap Uniswap V3 single-hop.
    ///
    /// L'astuce de l'offset :
    ///   exactInputSingle prend une struct en calldata. Après le sélecteur (4 bytes),
    ///   le layout ABI d'une struct passée en calldata est :
    ///
    ///     offset  field
    ///       0     tokenIn          (32 bytes)
    ///      32     tokenOut         (32 bytes)
    ///      64     fee              (32 bytes, uint24 padded)
    ///      96     recipient        (32 bytes)
    ///     128     amountIn         (32 bytes)  <-- c'est ici qu'on patch
    ///     160     amountOutMinimum (32 bytes)
    ///     192     sqrtPriceLimitX96(32 bytes)
    ///
    ///   amountInOffset = 128 (relatif au début du payload, SANS le sélecteur,
    ///   car dans le contrat on fait `add(add(stepData, 32), offset)` où stepData
    ///   est un bytes memory dont les 32 premiers bytes sont la longueur — donc
    ///   add(stepData, 32) pointe sur le sélecteur, et +128 atterrit sur amountIn).
    function _buildSingleHopSteps(
        address tokenIn,
        address tokenOut,
        uint24  fee
    ) internal view returns (Liquidator.SwapStep[] memory steps) {
        // On encode l'appel avec amountIn = 0 ; il sera patché au runtime
        bytes memory data = abi.encodeCall(
            ISwapRouter.exactInputSingle,
            ISwapRouter.ExactInputSingleParams({
                tokenIn:           tokenIn,
                tokenOut:          tokenOut,
                fee:               fee,
                recipient:         address(this), // sera l'adresse du liquidateur au runtime
                                                  // (on encode address(this) = adresse du test
                                                  //  mais le contrat appelle depuis lui-même)
                amountIn:          0,             // patché au runtime
                amountOutMinimum:  0,             // pas de slippage en test
                sqrtPriceLimitX96: 0
            })
        );

        // recipient doit être le contrat Liquidator lui-même pour qu'il reçoive les tokens.
        // Comme on ne connaît pas encore son adresse ici, on le patche aussi.
        // En pratique on peut le laisser à address(this) du test si on veut,
        // mais le plus propre est de déployer d'abord et patcher ensuite — voir _liquidate.

        // amountInOffset = 4 (selector) + 4*32 (tokenIn, tokenOut, fee, recipient) = 4 + 128 = 132
        // MAIS dans le contrat, stepData est un bytes memory donc :
        //   add(stepData, 32) => pointe sur le 1er byte du payload (= 1er byte du sélecteur)
        //   add(..., offset)  => saute `offset` bytes depuis là
        // Donc offset = 4 + 4*32 = 132 pour amountIn (le 5ème slot après le selector)
        uint256 amountInOffset = 4 + 4 * 32; // = 132

        steps = new Liquidator.SwapStep[](1);
        steps[0] = Liquidator.SwapStep({
            target:        UNI_ROUTER,
            data:          data,
            tokenIn:       tokenIn,
            tokenOut:      tokenOut,
            amountInOffset: amountInOffset
        });
    }

    function _liquidate(
        uint256                    blockNum,
        address                    loanToken,
        address                    collateralToken,
        address                    oracle,
        address                    irm,
        uint256                    lltv,
        address                    borrower,
        uint256                    seizedAssets,
        Liquidator.SwapStep[] memory steps
    ) internal {
        vm.rollFork(blockNum);

        Liquidator liquidator = new Liquidator(MORPHO);

        // Autoriser le router
        liquidator.setTarget(UNI_ROUTER, true);

        // Patcher le recipient dans le calldata de chaque step
        // recipient est au slot 3 (0-indexed) après le selector : offset = 4 + 3*32 = 100
        for (uint256 i = 0; i < steps.length; i++) {
            bytes memory d = steps[i].data;
            address liq = address(liquidator);
            uint256 recipientOffset = 4 + 3 * 32; // = 100
            assembly {
                // add(d, 32) => début du payload ; +recipientOffset => champ recipient
                mstore(add(add(d, 32), recipientOffset), liq)
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

        // Rendre le borrower liquidable : on réduit le prix oracle de 10 %
        uint256 price = IOracle(oracle).price();
        vm.mockCall(oracle, abi.encodeWithSignature("price()"), abi.encode(price * 90 / 100));

        liquidator.liquidate(mp, borrower, seizedAssets, 0, steps, 0);

        uint256 profit     = IERC20(loanToken).balanceOf(address(liquidator));
        uint256 profitColl = IERC20(collateralToken).balanceOf(address(liquidator));

        assertGt(profit + profitColl, 0, "no profit at all");
        console.log("profit multihop loanToken :", profit);
        console.log("profit multihop collToken :", profitColl);
    }
}
