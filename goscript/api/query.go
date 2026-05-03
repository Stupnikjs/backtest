package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// ── STRUCTS ──────────────────────────────────────────────────────────────────

type LiquidationsResponse struct {
	Transactions struct {
		Items []Transaction `json:"items"`
	} `json:"transactions"`
}

type Transaction struct {
	BlockNumber int64  `json:"blockNumber"`
	Hash        string `json:"hash"`
	Type        string `json:"type"`
	User        struct {
		Address string `json:"address"`
	} `json:"user"`
	Data LiquidationData `json:"data"`
}

type LiquidationData struct {
	SeizedAssets     json.Number `json:"seizedAssets"`
	SeizedAssetsUsd  json.Number `json:"seizedAssetsUsd"`
	RepaidAssets     json.Number `json:"repaidAssets"`
	RepaidAssetsUsd  json.Number `json:"repaidAssetsUsd"`
	BadDebtAssetsUsd json.Number `json:"badDebtAssetsUsd"`
	Liquidator       string      `json:"liquidator"`
	Market           struct {
		UniqueKey string `json:"uniqueKey"`
		LoanAsset struct {
			Symbol string `json:"symbol"`
		} `json:"loanAsset"`
		CollateralAsset struct {
			Symbol string `json:"symbol"`
		} `json:"collateralAsset"`
	} `json:"market"`
}

type MarketResponse struct {
	MarketByUniqueKey Market `json:"marketByUniqueKey"`
}

type Market struct {
	LLTV            json.Number `json:"lltv"`
	OracleAddress   string      `json:"oracleAddress"`
	IRMAddress      string      `json:"irmAddress"`
	LoanAsset       Asset       `json:"loanAsset"`
	CollateralAsset Asset       `json:"collateralAsset"`
}

type Asset struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// ── QUERY ────────────────────────────────────────────────────────────────────

func GetLiquidations(ctx context.Context, chainID int, count int) ([]Transaction, error) {
	query := fmt.Sprintf(`{
		transactions(
			first: %d
			orderBy: Timestamp
			orderDirection: Desc
			where: {
				type_in: [MarketLiquidation]
				chainId_in: [%d]
			}
		) {
			items {
				blockNumber
				hash
				type
				user { address }
				data {
					... on MarketLiquidationTransactionData {
						seizedAssets
						seizedAssetsUsd
						repaidAssets
						repaidAssetsUsd
						badDebtAssetsUsd
						liquidator
						market {
							uniqueKey
							loanAsset { symbol }
							collateralAsset { symbol }
						}
					}
				}
			}
		}
	}`, count, chainID)

	var out LiquidationsResponse
	if err := Query(ctx, query, &out); err != nil {
		return nil, fmt.Errorf("get liquidations: %w", err)
	}
	return out.Transactions.Items, nil
}

func GetMarketByUniqueKey(ctx context.Context, uniqueKey string, chainID int) (*Market, error) {
	query := fmt.Sprintf(`{
		marketByUniqueKey(
			uniqueKey: "%s"
			chainId: %d
		) {
			lltv
			oracleAddress
			irmAddress
			loanAsset {
				address
				symbol
				decimals
			}
			collateralAsset {
				address
				symbol
				decimals
			}
		}
	}`, uniqueKey, chainID)

	var out MarketResponse
	if err := Query(ctx, query, &out); err != nil {
		return nil, fmt.Errorf("get market by unique key: %w", err)
	}
	return &out.MarketByUniqueKey, nil
}
