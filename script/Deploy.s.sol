// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../src/LiquidatorMulti.sol";

contract DeployLiquidatorMulti is Script {
    address constant MORPHO = 0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb;

    function run() external {
        uint256 deployerPrivateKey = vm.envUint("BASE_PK");
        vm.startBroadcast(deployerPrivateKey);

        LiquidatorMulti liquidator = new LiquidatorMulti(MORPHO);
        console.log("LiquidatorMulti deployed at:", address(liquidator));

        vm.stopBroadcast();
    }
}