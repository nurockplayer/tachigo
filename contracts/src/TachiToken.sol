// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/// @title TachiToken
/// @notice Soulbound ERC-20；持有者不可自由轉讓。
/// @dev MVP 由 owner 執行 mint/burn；未來若需要協議內消費或結算，將在 Phase 2 透過 Router Contract 設計。
contract TachiToken is ERC20, Ownable {
    uint256 public constant MAX_SUPPLY = 1_000_000_000 * 10 ** 18; // 10 億枚，單位為 wei（最小單位）

    constructor() ERC20("Tachi", "TACHI") Ownable(msg.sender) {}

    /// @notice 鑄造代幣（僅限 owner）。
    function mint(address to, uint256 amount) external onlyOwner {
        require(totalSupply() + amount <= MAX_SUPPLY, "TachiToken: cap exceeded");
        _mint(to, amount);
    }

    /// @notice 銷毀代幣（消費路徑，僅限 owner）。
    function burn(address from, uint256 amount) external onlyOwner {
        _burn(from, amount);
    }

    /// @dev Soulbound：禁止持有者自由轉帳。
    function transfer(address, uint256) public pure override returns (bool) {
        revert("TachiToken: soulbound");
    }

    /// @dev Soulbound：禁止持有者自由轉帳。
    function transferFrom(address, address, uint256) public pure override returns (bool) {
        revert("TachiToken: soulbound");
    }

    /// @dev Soulbound：禁止建立 allowance，避免產生永遠無法執行的授權假象。
    function approve(address, uint256) public pure override returns (bool) {
        revert("TachiToken: soulbound");
    }
}
