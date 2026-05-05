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
}