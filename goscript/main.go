package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/Stupnikjs/backtest/anvil"
	"github.com/Stupnikjs/backtest/api"
	"github.com/Stupnikjs/backtest/contract"
	"github.com/Stupnikjs/backtest/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
)

var ARBUniRouter = common.HexToAddress("0xkopkpo")

/*
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

*/

type LiquidatePos struct {
	Borrower     common.Address
	SeizedAsset  *big.Int
	RepaidShares *big.Int
	SwapRouter   common.Address
	PoolFee      *big.Int
	MinOut       *big.Int
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}
	ctx := context.Background()
	liquidations, err := api.GetLiquidations(ctx, 8453, 200)
	if err != nil {
		log.Fatal(err)
	}
	for i, tx := range liquidations {
		fmt.Printf("[%d] %s | %s/%s | seized: $%.2f\n",
			tx.BlockNumber,
			tx.Hash,
			tx.Data.Market.CollateralAsset.Symbol,
			tx.Data.Market.LoanAsset.Symbol,
			tx.Data.SeizedAssetsUsd,
		)

		mp, err := api.GetMarketByUniqueKey(ctx, tx.Data.Market.UniqueKey, 8453)
		if err != nil {
			fmt.Println(err)
		}
		marketContractParams := contract.MarketContractParams{
			LoanToken:       common.HexToAddress(mp.LoanAsset.Address),
			CollateralToken: common.HexToAddress(mp.CollateralAsset.Address),
			Oracle:          common.HexToAddress(mp.OracleAddress),
			Irm:             common.HexToAddress(mp.IRMAddress),
			Lltv:            utils.ParseBigInt(mp.LLTV.String()),
		}
		pos := LiquidatePos{
			Borrower:     common.HexToAddress(tx.User.Address),
			SeizedAsset:  utils.ParseBigInt(tx.Data.SeizedAssets.String()),
			RepaidShares: big.NewInt(0),
			SwapRouter:   ARBUniRouter,
			PoolFee:      big.NewInt(100),
			MinOut:       big.NewInt(0),
		}
		// fork then simulate liquidation with contract
		if tx.Data.Market.CollateralAsset.Symbol == "USDC" && tx.Data.Market.LoanAsset.Symbol == "WETH" {
			backtest(uint64(tx.BlockNumber), 8450+i, marketContractParams, pos)
		} else if tx.Data.Market.CollateralAsset.Symbol == "WETH" && tx.Data.Market.LoanAsset.Symbol == "USDC" {

		}

	}
}

func backtest(blockNum uint64, port int, marketParams contract.MarketContractParams, pos LiquidatePos) {
	rpc := os.Getenv("ARB_RPC")
	if rpc == "" {
		fmt.Println("ARB_RPC not set")
		return
	}

	anvilInstance, err := anvil.StartAnvil(rpc, blockNum, port)
	if err != nil {
		fmt.Printf("  anvil start failed (block=%d port=%d): %v\n", blockNum, port, err)
		return
	}
	defer anvilInstance.Stop()

	fmt.Printf("  anvil running on port %d for block %d \n", port, blockNum)

	// TODO: deploy/connect contract, call liquidate with marketParams + pos
	_ = marketParams
	_ = pos
}
