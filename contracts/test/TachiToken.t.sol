// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/TachiToken.sol";

contract TachiTokenTest is Test {
    TachiToken token;
    address alice;
    address bob;

    function setUp() public {
        alice = address(0xA1);
        bob   = address(0xB0);
        token = new TachiToken();
    }

    // --- mint ---

    function test_mint_owner_succeeds() public {
        token.mint(alice, 1_000e18);
        assertEq(token.balanceOf(alice), 1_000e18);
    }

    function test_mint_nonOwner_reverts() public {
        vm.prank(alice);
        vm.expectRevert();
        token.mint(bob, 1_000e18);
    }

    function test_mint_exceedsCap_reverts() public {
        uint256 cap = 1_000_000_000e18;
        token.mint(alice, cap);
        vm.expectRevert();
        token.mint(alice, 1);
    }

    // --- transfer (Soulbound) ---

    function test_transfer_reverts() public {
        token.mint(alice, 1_000e18);
        vm.prank(alice);
        vm.expectRevert();
        token.transfer(bob, 100e18);
    }

    function test_transferFrom_reverts() public {
        token.mint(alice, 1_000e18);
        vm.prank(alice);
        token.approve(bob, 100e18);
        vm.prank(bob);
        vm.expectRevert();
        token.transferFrom(alice, bob, 100e18);
    }

    // --- burn ---

    function test_burn_owner_succeeds() public {
        token.mint(alice, 1_000e18);
        token.burn(alice, 400e18);
        assertEq(token.balanceOf(alice), 600e18);
    }

    // --- zero address ---

    function test_mint_zeroAddress_reverts() public {
        vm.expectRevert();
        token.mint(address(0), 1_000e18);
    }

    // --- burn edge ---

    function test_burn_zeroAmount_succeeds() public {
        token.mint(alice, 1_000e18);
        token.burn(alice, 0);
        assertEq(token.balanceOf(alice), 1_000e18);
    }

    // --- approve ---

    function test_approve_succeeds_even_though_transfer_blocked() public {
        token.mint(alice, 1_000e18);
        vm.prank(alice);
        token.approve(bob, 500e18);
        assertEq(token.allowance(alice, bob), 500e18);
    }
}
