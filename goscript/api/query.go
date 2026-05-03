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
