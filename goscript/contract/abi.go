package contract

import "github.com/lmittmann/w3"

var FuncLiquidate = w3.MustNewFunc(
	`liquidate(
			(address loanToken, address collateralToken, address oracle, address irm, uint256 lltv) marketParams,
			address borrower,
			uint256 seizedAssets,
			uint256 repaidShares,
			address swapRouter,
			uint24 poolFee,
			uint256 minOut
		)`,
	``,
)
