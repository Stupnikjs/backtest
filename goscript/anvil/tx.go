package anvil

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/Stupnikjs/backtest/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/lmittmann/w3/module/eth"
	"github.com/lmittmann/w3/w3types"
)

// ── TYPES ────────────────────────────────────────────────────────────────────

type Signer struct {
	key    *ecdsa.PrivateKey
	signer types.Signer
}

type TxParams struct {
	To       *common.Address
	Calldata []byte
	Value    *big.Int
}

type LiquidateArgs struct {
	MarketParams interface{}
	Borrower     common.Address
	SeizedAssets *big.Int
	RepaidShares *big.Int
	SwapRouter   common.Address
	PoolFee      *big.Int
	MinOut       *big.Int
}

// ── SIGNER ───────────────────────────────────────────────────────────────────

// AnvilSigner utilise le compte 0 d'Anvil par défaut
func NewAnvilSigner(chainid int64) (*Signer, error) {

	keyHex := os.Getenv("BASE_PK")
	if keyHex == "" {
		return nil, fmt.Errorf("LIQUIDATOR__BASE_PRIVATE_KEY not set")
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return &Signer{
		key:    key,
		signer: types.NewLondonSigner(big.NewInt(chainid)),
	}, nil
}

// ── TX ───────────────────────────────────────────────────────────────────────

func (a *AnvilInstance) SendSignedTx(ctx context.Context, params TxParams) (common.Hash, error) {
	sender := crypto.PubkeyToAddress(a.Signer.key.PublicKey)

	var nonce uint64
	var gasPrice *big.Int
	var gasEst uint64

	msg := w3types.Message{
		From:  sender,
		To:    params.To,
		Input: params.Calldata,
		Value: params.Value,
	}

	if err := a.Client.CallCtx(ctx,
		eth.Nonce(sender, nil).Returns(&nonce),
		eth.GasPrice().Returns(&gasPrice),
		eth.EstimateGas(&msg, nil).Returns(&gasEst),
	); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: fetch params: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		To:        params.To,
		Data:      params.Calldata,
		Value:     params.Value,
		Gas:       gasEst * 12 / 10,
		GasTipCap: big.NewInt(1e9),
		GasFeeCap: new(big.Int).Add(gasPrice, big.NewInt(1e9)),
	})

	chainID := big.NewInt(8453) // Base, adapte selon ton fork
	signer := types.LatestSignerForChainID(chainID)
	signedTx, err := types.SignTx(tx, signer, a.Signer.key)
	if err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: sign: %w", err)
	}

	var txHash common.Hash
	if err := a.Client.CallCtx(ctx, eth.SendTx(signedTx).Returns(&txHash)); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: send: %w", err)
	}

	return txHash, nil
}

// ── LIQUIDATE ────────────────────────────────────────────────────────────────

func (a *AnvilInstance) LiquidateCall(ctx context.Context, args LiquidateArgs, privateKey *ecdsa.PrivateKey, liquidatorAddr common.Address) error {
	calldata, err := contract.FuncLiquidate.EncodeArgs(
		args.MarketParams,
		args.Borrower,
		args.SeizedAssets,
		args.RepaidShares,
		args.SwapRouter,
		args.PoolFee,
		args.MinOut,
	)
	if err != nil {
		return fmt.Errorf("LiquidateCall: encode: %w", err)
	}

	txHash, err := a.SendSignedTx(ctx, TxParams{
		To:       &liquidatorAddr,
		Calldata: calldata,
		Value:    big.NewInt(0),
	})
	if err != nil {
		return fmt.Errorf("LiquidateCall: %w", err)
	}

	fmt.Printf("[liquidate] tx: %s\n", txHash.Hex())
	return nil
}

func (a *AnvilInstance) DeployContract(ctx context.Context, bytecode []byte) (common.Address, error) {
	txHash, err := a.SendSignedTx(ctx, TxParams{
		To:       nil, // nil = déploiement
		Calldata: bytecode,
		Value:    big.NewInt(0),
	})
	if err != nil {
		return common.Address{}, fmt.Errorf("deploy: %w", err)
	}

	// Récupérer l'adresse du contrat depuis le receipt
	var receipt *types.Receipt
	if err := a.Client.CallCtx(ctx,
		eth.TxReceipt(txHash).Returns(&receipt),
	); err != nil {
		return common.Address{}, fmt.Errorf("deploy receipt: %w", err)
	}

	return receipt.ContractAddress, nil
}
