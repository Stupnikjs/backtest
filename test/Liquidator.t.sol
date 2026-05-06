pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/Liquidator.sol";

interface IOracle {
    function price() external view returns (uint256);
}

contract LiquidatorTest is Test {
    address constant MORPHO     = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;
    address constant UNI_ROUTER = 0x2626664c2603336E57B271c5C0b26F421741e481;

    function setUp() public {
        vm.createSelectFork(vm.envString("BASE_RPC"));
    }

    function test_WETH_USDC() public {
        _liquidate({
            blockNum:        45433873,
            loanToken:       0x4200000000000000000000000000000000000006,
            collateralToken: 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913,
            oracle:          0xD09048c8B568Dbf5f189302beA26c9edABFC4858,
            irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
            lltv:            860000000000000000,
            borrower:        0x924D53C12e04B7F74D97c9770889502D7aecc95e,
            seizedAssets:    8984643,
            poolFee:         500
        });
    }

    function test_USDC_WETH() public {
        _liquidate({
            blockNum:        45379997,
            loanToken:       0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913,
            collateralToken: 0x4200000000000000000000000000000000000006,
            oracle:          0xFEa2D58cEfCb9fcb597723c6bAE66fFE4193aFE4,
            irm:             0x46415998764C29aB2a25CbeA6254146D50D22687,
            lltv:            860000000000000000,
            borrower:        0xb8Aa8a5E09245c6D4FAf602020853dE595d9f14A,
            seizedAssets:    4624239072469418,
            poolFee:         500
        });
    }

    function _liquidate(
        uint256 blockNum,
        address loanToken,
        address collateralToken,
        address oracle,
        address irm,
        uint256 lltv,
        address borrower,
        uint256 seizedAssets,
        uint24  poolFee
    ) internal {
        vm.rollFork(blockNum);

        Liquidator liquidator = new Liquidator(MORPHO);

        MarketParams memory mp = MarketParams({
            loanToken:       loanToken,
            collateralToken: collateralToken,
            oracle:          oracle,
            irm:             irm,
            lltv:            lltv
        });

        uint256 price = IOracle(oracle).price();
        vm.mockCall(oracle, abi.encodeWithSignature("price()"), abi.encode(price * 90 / 100));

        liquidator.liquidate(mp, borrower, seizedAssets, 0, UNI_ROUTER, poolFee, 0);

        uint256 profit = IERC20(loanToken).balanceOf(address(liquidator));
        uint256 profitColl = IERC20(collateralToken).balanceOf(address(liquidator));
        assertGt(profit, 0, "no profit");
        console.log("profit:", profit);
        console.log("profitcoll:", profitColl);
    }
}