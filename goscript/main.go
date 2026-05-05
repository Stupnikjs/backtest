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

// var ARBUniRouter = common.HexToAddress("0xe592427a0aece92de3edee1f18e0157c05861564")
var BaseUniRouter = common.HexToAddress("0x2626664c2603336E57B271c5C0b26F421741e481")
var MorphoBlueAddr = common.HexToAddress("0x6c247b1F6182318877311737BaC0844bAa518F5e")

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
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println(err)
	}
	ctx := context.Background()
	liquidations, err := api.GetLiquidations(ctx, 8453, 200)
	if err != nil {
		log.Fatal(err)
	}
	for i, tx := range liquidations {

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
			SwapRouter:   BaseUniRouter,
			PoolFee:      big.NewInt(500),
			MinOut:       big.NewInt(0),
		}
		// fork then simulate liquidation with contract
		if tx.Data.Market.CollateralAsset.Symbol == "USDC" && tx.Data.Market.LoanAsset.Symbol == "WETH" {
			backtest(uint64(tx.BlockNumber), 8450+i, marketContractParams, pos)
		} else if tx.Data.Market.CollateralAsset.Symbol == "WETH" && tx.Data.Market.LoanAsset.Symbol == "USDC" {
			backtest(uint64(tx.BlockNumber), 8450+i, marketContractParams, pos)
		}

	}
}

func backtest(blockNum uint64, port int, marketParams contract.MarketContractParams, pos LiquidatePos) {
	ctx := context.Background()
	rpc := os.Getenv("BASE_RPC")
	fmt.Println("RPC BASE :", rpc)
	anvilInstance, err := anvil.StartAnvil(rpc, blockNum, port, 8453)
	if err != nil {
		fmt.Printf("  anvil start failed (block=%d port=%d): %v\n", blockNum, port, err)
		return
	}
	defer anvilInstance.Stop()

	fmt.Printf("  anvil running on port %d for block %d \n", port, blockNum)

	// TODO: deploy/connect contract, call liquidate with marketParams + pos
	bytecode, err := contract.LoadBytecode("../out/Liquidator.sol/Liquidator.json")
	if err != nil {
		fmt.Println(err)
	}
	MorphoBlueAddr = common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")
	bytecode, err = contract.EncodedBytecodeWithConstructor(bytecode, MorphoBlueAddr)
	if err != nil {
		fmt.Println(err)
	}

	liquidatorAddress, err := anvilInstance.DeployContract(ctx, bytecode)
	if err != nil {
		fmt.Printf("deploy failed, aborting: %w", err)
		return
	}
	if liquidatorAddress == (common.Address{}) {
		fmt.Printf("deploy returned zero address, aborting")
		return
	}

	args := contract.LiquidateArgs{
		MarketParams: marketParams,
		Borrower:     pos.Borrower,
		SeizedAssets: pos.SeizedAsset,
		RepaidShares: pos.RepaidShares,
		SwapRouter:   pos.SwapRouter,
		PoolFee:      pos.PoolFee,
		MinOut:       pos.MinOut,
	}

	tx, err := anvilInstance.LiquidateCall(ctx, args, liquidatorAddress)
	_ = tx
	if err != nil {
		fmt.Printf("liquidate failed: %v\n", err)

	}

}
