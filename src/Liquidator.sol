// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {IMorpho, MarketParams} from "morpho-blue/src/interfaces/IMorpho.sol";
import {SafeERC20} from "openzeppelin-contracts/contracts/token/ERC20/utils/SafeERC20.sol";
import {MorphoLib} from "morpho-blue/src/libraries/periphery/MorphoLib.sol";
import {MarketParamsLib} from "morpho-blue/src/libraries/MarketParamsLib.sol";
import {IERC20} from "openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";

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

contract Liquidator {
    using SafeERC20 for IERC20;
    using MarketParamsLib for MarketParams;
    using MorphoLib for IMorpho;

    error NotOwner();
    error NotMorpho();
    error NotInLiquidation();
    error ZeroLiquidationParams();
    error ZeroCollateralBalance();

    IMorpho public immutable MORPHO;
    address public immutable owner;
    bool private _inLiquidation;

    struct LiquidationData {
        MarketParams marketParams;
        address swapRouter;
        uint24  poolFee;
        uint256 minOut;
    }

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier onlyMorpho() {
        if (msg.sender != address(MORPHO)) revert NotMorpho();
        if (!_inLiquidation)              revert NotInLiquidation();
        _;
    }

    constructor(address morpho) {
        MORPHO = IMorpho(morpho);
        owner  = msg.sender;
    }

    function liquidate(
        MarketParams calldata marketParams,
        address borrower,
        uint256 seizedAssets,
        uint256 repaidShares,
        address swapRouter,
        uint24  poolFee,
        uint256 minOut
    ) external onlyOwner {
        if (seizedAssets == 0 && repaidShares == 0) revert ZeroLiquidationParams();

        IERC20(marketParams.loanToken).forceApprove(address(MORPHO), type(uint256).max);

        bytes memory callbackData = abi.encode(LiquidationData({
            marketParams: marketParams,
            swapRouter:   swapRouter,
            poolFee:      poolFee,
            minOut:       minOut
        }));

        _inLiquidation = true;
        try MORPHO.liquidate(marketParams, borrower, seizedAssets, repaidShares, callbackData) {
            _inLiquidation = false;
        } catch (bytes memory err) {
            _inLiquidation = false;
            assembly { revert(add(err, 32), mload(err)) }
        }
    }

    function onMorphoLiquidate(uint256 /*repaidAssets*/, bytes calldata data) external onlyMorpho {
        LiquidationData memory d = abi.decode(data, (LiquidationData));

        address collateral = d.marketParams.collateralToken;
        address loanToken  = d.marketParams.loanToken;

        uint256 amountIn = IERC20(collateral).balanceOf(address(this));
        if (amountIn == 0) revert ZeroCollateralBalance();

        IERC20(collateral).forceApprove(d.swapRouter, amountIn);

        ISwapRouter(d.swapRouter).exactInputSingle(
            ISwapRouter.ExactInputSingleParams({
                tokenIn:           collateral,
                tokenOut:          loanToken,
                fee:               d.poolFee,
                recipient:         address(this),
                amountIn:          amountIn,
                amountOutMinimum:  d.minOut,
                sqrtPriceLimitX96: 0
            })
        );
    }

    function sweep(address token) external onlyOwner {
        uint256 bal = IERC20(token).balanceOf(address(this));
        IERC20(token).safeTransfer(owner, bal);
    }

    receive() external payable {}
}