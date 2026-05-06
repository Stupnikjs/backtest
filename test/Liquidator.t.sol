// test/Liquidator.t.sol
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/Liquidator.sol";


interface IOracle {
    function price() external view returns (uint256);
}
contract LiquidatorTest is Test {
    Liquidator liquidator;
    
    address constant MORPHO = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;
    address constant UNI_ROUTER = 0x2626664c2603336E57B271c5C0b26F421741e481;
    
    function setUp() public {
        // Fork Base au bloc exact de la liquidation historique - 1
        vm.createSelectFork(vm.envString("BASE_RPC"), 45433873);
        liquidator = new Liquidator(MORPHO);
    }

   function test_liquidate_WETH_USDC() public {
    // 1. Fork au bon bloc
    vm.rollFork(45433873);
    
    // 2. Deploy après le rollFork
    liquidator = new Liquidator(MORPHO);

    MarketParams memory mp = MarketParams({
        loanToken:       0x4200000000000000000000000000000000000006,
        collateralToken: 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913,
        oracle:          0xD09048c8B568Dbf5f189302beA26c9edABFC4858,
        irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
        lltv:            860000000000000000
    });

    address borrower = 0x924D53C12e04B7F74D97c9770889502D7aecc95e;
       // Prix actuel de l'oracle
    uint256 currentPrice = IOracle(mp.oracle).price();
    
    // Baisse le prix de 10% pour rendre la position liquidatable
    // Mock l'oracle pour retourner un prix plus bas
    vm.mockCall(
        mp.oracle,
        abi.encodeWithSignature("price()"),
        abi.encode(currentPrice * 90 / 100)
    );
    liquidator.liquidate(mp, borrower, 8984643, 0, UNI_ROUTER, 500, 0);

    uint256 bal = IERC20(mp.loanToken).balanceOf(address(liquidator));
    assertGt(bal, 0, "no profit");
    console.log("profit WETH:", bal);
}



// ─── 1. REVERT CASES ───────────────────────────────────────

    function test_revert_notLiquidatable() public {
        // Sans mock du prix → position healthy → doit revert
        vm.expectRevert();
        liquidator.liquidate(mp, borrower, 8984643, 0, UNI_ROUTER, 500, 0);
    }

    function test_revert_wrongCaller() public {
        // Si ton contrat a un onlyOwner ou access control
        vm.prank(address(0xdead));
        vm.expectRevert();
        liquidator.liquidate(mp, borrower, 8984643, 0, UNI_ROUTER, 500, 0);
    }

    function test_revert_slippageTooHigh() public {
        _mockPriceDown(10);
        // amountOutMinimum astronomique → swap revert
        uint256 impossibleMinOut = type(uint256).max;
        vm.expectRevert();
        liquidator.liquidate(mp, borrower, 8984643, impossibleMinOut, UNI_ROUTER, 500, 0);
    }

    // ─── 2. DIFFÉRENTS MARKETS ─────────────────────────────────

    function test_liquidate_cbBTC_USDC() public {
        vm.rollFork(SOME_BLOCK);
        MarketParams memory mp2 = MarketParams({
            loanToken:       USDC,
            collateralToken: cbBTC,
            oracle:          0x...,
            irm:             IRM,
            lltv:            860000000000000000
        });
        _mockPriceDown(10);
        liquidator.liquidate(mp2, borrower2, seizedAmount, 0, UNI_ROUTER, 3000, 0);
        assertGt(IERC20(USDC).balanceOf(address(liquidator)), 0);
    }

    // ─── 3. FEE TIERS ──────────────────────────────────────────

    function test_liquidate_feeTier_100() public { ... }
    function test_liquidate_feeTier_500() public { ... }  // ton test actuel
    function test_liquidate_feeTier_3000() public { ... }

    // ─── 4. EDGE CASES MONTANTS ────────────────────────────────

    function test_liquidate_maxSeizable() public {
        // seize le max autorisé par Morpho
        _mockPriceDown(10);
        liquidator.liquidate(mp, borrower, type(uint256).max, 0, UNI_ROUTER, 500, 0);
    }

    function test_liquidate_dustAmount() public {
        // tout petit montant → toujours profitable après gas ?
        _mockPriceDown(10);
        liquidator.liquidate(mp, borrower, 1, 0, UNI_ROUTER, 500, 0);
    }

    // ─── 5. PROFIT ASSERTIONS ──────────────────────────────────

    function test_profit_increases_with_discount() public {
        uint256 profit10 = _liquidateAndGetProfit(10); // -10%
        uint256 profit20 = _liquidateAndGetProfit(20); // -20%
        assertGt(profit20, profit10, "bigger discount = more profit");
    }

    // ─── 6. RESCUE / ADMIN ─────────────────────────────────────

    function test_rescue_erc20() public {
        // Si tu as une fonction rescue sur ton contrat
        deal(USDC, address(liquidator), 1000e6);
        liquidator.rescue(USDC, owner, 1000e6);
        assertEq(IERC20(USDC).balanceOf(owner), 1000e6);
    }

    // ─── HELPERS ───────────────────────────────────────────────

    function _mockPriceDown(uint256 pct) internal {
        uint256 current = IOracle(mp.oracle).price();
        vm.mockCall(
            mp.oracle,
            abi.encodeWithSignature("price()"),
            abi.encode(current * (100 - pct) / 100)
        );
    }

    function _liquidateAndGetProfit(uint256 pctDown) internal returns (uint256) {
        _mockPriceDown(pctDown);
        liquidator.liquidate(mp, borrower, 8984643, 0, UNI_ROUTER, 500, 0);
        return IERC20(mp.loanToken).balanceOf(address(liquidator));
    }
}