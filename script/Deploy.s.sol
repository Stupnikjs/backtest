// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../src/Liquidator.sol";



/*
forge script script/DeployLiquidator.s.sol:DeployLiquidatorMulti \
  --rpc-url $BASE_RPC_URL \
  --broadcast \
  --verify


  0x6c247b1F6182318877311737BaC0844bAa518F5e arbitrum morpho
  0xD50F2DffFd62f94Ee4AEd9ca05C61d0753268aBc katana
*/



contract DeployLiquidatorMulti is Script {
    address constant MORPHO = 0xD50F2DffFd62f94Ee4AEd9ca05C61d0753268aBc;

    function run() external {
        uint256 deployerPrivateKey = vm.envUint("BASE_PK");
        vm.startBroadcast(deployerPrivateKey);

        Liquidator liquidator = new Liquidator(MORPHO);
        console.log("LiquidatorMulti deployed at:", address(liquidator));

        vm.stopBroadcast();
    }
}