package contract

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type MarketContractParams struct {
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	Irm             common.Address
	Lltv            *big.Int
}
