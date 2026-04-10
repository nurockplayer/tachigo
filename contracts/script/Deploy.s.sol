// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../src/TachiToken.sol";

contract Deploy is Script {
    function run() external returns (TachiToken token) {
        uint256 deployerPrivateKey = vm.envUint("DEPLOYER_PRIVATE_KEY");
        vm.startBroadcast(deployerPrivateKey);
        token = new TachiToken();
        vm.stopBroadcast();
        console.log("TachiToken deployed at:", address(token));
    }
}
