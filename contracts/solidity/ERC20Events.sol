// SPDX-License-Identifier: MIT
// OpenZeppelin Contracts (last updated v4.6.0) (token/ERC20/IERC20.sol)

pragma solidity ^0.8.0;

contract ERC20Events {
    uint8 private _decimals;

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);

    /**
    * @dev Sets `_decimals` as `decimals_ once at Deployment'
    */
    function setupDecimals(uint8 decimals_) external {
        _decimals = decimals_;
    }
}