package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Stupnikjs/backtest/api"
)

func main() {
	ctx := context.Background()
	liquidations, err := api.GetLiquidations(ctx, 8453, 20)
	if err != nil {
		log.Fatal(err)
	}
	for _, tx := range liquidations {
		fmt.Printf("[%d] %s | %s/%s | seized: $%.2f\n",
			tx.BlockNumber,
			tx.Hash,
			tx.Data.Market.CollateralAsset.Symbol,
			tx.Data.Market.LoanAsset.Symbol,
			tx.Data.SeizedAssetsUsd,
		)
		// fork then simulate liquidation with contract

 // filter les marché liquide uniswap v3

	}
}
