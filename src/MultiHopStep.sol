// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {IMorpho, MarketParams} from "morpho-blue/src/interfaces/IMorpho.sol";
import {SafeERC20} from "openzeppelin-contracts/contracts/token/ERC20/utils/SafeERC20.sol";
import {MorphoLib} from "morpho-blue/src/libraries/periphery/MorphoLib.sol";
import {MarketParamsLib} from "morpho-blue/src/libraries/MarketParamsLib.sol";
import {IERC20} from "openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";

contract Liquidator {
    using SafeERC20 for IERC20;
    using MarketParamsLib for MarketParams;
    using MorphoLib for IMorpho;

    // ─────────────────────────────────────────────
    // Errors
    // ─────────────────────────────────────────────

    error NotOwner();
    error NotMorpho();
    error NotInLiquidation();
    error ZeroLiquidationParams();
    error EmptyPipeline();
    error StepZeroInput(uint256 stepIndex);
    error StepZeroOutput(uint256 stepIndex);
    error StepCallFailed(uint256 stepIndex);
    error TargetNotAllowed(address target);
    error InsufficientToRepay(uint256 amountOut, uint256 repaidAssets);
    error BelowMinOut(uint256 amountOut, uint256 minOut);

    // ─────────────────────────────────────────────
    // Structs
    // ─────────────────────────────────────────────

    struct SwapStep {
        address target;
        bytes   data;
        address tokenIn;
        address tokenOut;
        uint256 amountInOffset; // offset dans data pour patcher amountIn, 0 = pas de patch
    }

    struct LiquidationData {
        MarketParams marketParams;
        SwapStep[]   steps;
        uint256      minOut;
    }

    // ─────────────────────────────────────────────
    // State
    // ─────────────────────────────────────────────

    IMorpho public immutable MORPHO;
    address public immutable owner;
    bool    private _inLiquidation;

    mapping(address => bool) public allowedTargets;

    // ─────────────────────────────────────────────
    // Modifiers
    // ─────────────────────────────────────────────

    modifier onlyOwner() {
        if (msg.sender != owner) revert NotOwner();
        _;
    }

    modifier onlyMorpho() {
        if (msg.sender != address(MORPHO)) revert NotMorpho();
        if (!_inLiquidation)               revert NotInLiquidation();
        _;
    }

    // ─────────────────────────────────────────────
    // Constructor
    // ─────────────────────────────────────────────

    constructor(address morpho) {
        MORPHO = IMorpho(morpho);
        owner  = msg.sender;
    }

    // ─────────────────────────────────────────────
    // Admin
    // ─────────────────────────────────────────────

    function setTarget(address target, bool allowed) external onlyOwner {
        allowedTargets[target] = allowed;
    }

    function sweep(address token) external onlyOwner {
        uint256 bal = IERC20(token).balanceOf(address(this));
        IERC20(token).safeTransfer(owner, bal);
    }

    // ─────────────────────────────────────────────
    // Liquidation
    // ─────────────────────────────────────────────

    function liquidate(
        MarketParams calldata marketParams,
        address               borrower,
        uint256               seizedAssets,
        uint256               repaidShares,
        SwapStep[] calldata   steps,
        uint256               minOut
    ) external onlyOwner {
        if (seizedAssets == 0 && repaidShares == 0) revert ZeroLiquidationParams();
        if (steps.length == 0)                      revert EmptyPipeline();

        IERC20(marketParams.loanToken).forceApprove(address(MORPHO), type(uint256).max);

        bytes memory callbackData = abi.encode(LiquidationData({
            marketParams: marketParams,
            steps:        steps,
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

    // ─────────────────────────────────────────────
    // Morpho callback
    // ─────────────────────────────────────────────

    function onMorphoLiquidate(uint256 repaidAssets, bytes calldata data) external onlyMorpho {
        LiquidationData memory d = abi.decode(data, (LiquidationData));

        uint256 len = d.steps.length;

        for (uint256 i = 0; i < len; i++) {
            SwapStep memory step = d.steps[i];

            if (!allowedTargets[step.target]) revert TargetNotAllowed(step.target);

            uint256 amountIn = IERC20(step.tokenIn).balanceOf(address(this));
            if (amountIn == 0) revert StepZeroInput(i);

            IERC20(step.tokenIn).forceApprove(step.target, amountIn);

            // Patch amountIn dans le calldata au runtime
            bytes memory stepData = step.data;
            if (step.amountInOffset != 0) {
                assembly {
                    // store le nouveau amount in a l'offset
                    mstore(add(stepData, step.amountInOffset), amountIn)
                }
            }

            uint256 balBefore = IERC20(step.tokenOut).balanceOf(address(this));

            (bool ok, bytes memory reason) = step.target.call(stepData);
            if (!ok) _revertWithStepError(i, reason);

            IERC20(step.tokenIn).forceApprove(step.target, 0);

            uint256 received = IERC20(step.tokenOut).balanceOf(address(this)) - balBefore;
            if (received == 0) revert StepZeroOutput(i);
        }

        address loanToken = d.marketParams.loanToken;
        uint256 amountOut = IERC20(loanToken).balanceOf(address(this));

        if (amountOut < repaidAssets) revert InsufficientToRepay(amountOut, repaidAssets);
        if (amountOut < d.minOut)     revert BelowMinOut(amountOut, d.minOut);
    }

    // ─────────────────────────────────────────────
    // Internals
    // ─────────────────────────────────────────────

    function _revertWithStepError(uint256 stepIndex, bytes memory reason) internal pure {
        if (reason.length > 0) {
            assembly { revert(add(reason, 32), mload(reason)) }
        }
        revert StepCallFailed(stepIndex);
    }

    receive() external payable {}
}