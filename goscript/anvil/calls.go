package anvil

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Stupnikjs/backtest/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
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

// ── SIGNER ───────────────────────────────────────────────────────────────────

const AnvilPrivateKey0 = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

// AnvilSigner utilise le compte 0 d'Anvil par défaut
func NewAnvilSigner(chainid int64) (*Signer, error) {

	key, err := crypto.HexToECDSA(strings.TrimPrefix(AnvilPrivateKey0, "0x"))
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
	// transaction sender wallet
	sender := crypto.PubkeyToAddress(a.Signer.key.PublicKey)

	var nonce uint64
	var gasPrice *big.Int

	if err := a.Client.CallCtx(ctx,
		eth.Nonce(sender, nil).Returns(&nonce),
		eth.GasPrice().Returns(&gasPrice),
	); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: fetch params: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		To:        params.To,
		Data:      params.Calldata,
		Value:     params.Value,
		Gas:       3_000_000, // hardcodé, assez large pour deploy + liquidation
		GasTipCap: big.NewInt(1e9),
		GasFeeCap: new(big.Int).Add(gasPrice, big.NewInt(1e9)),
	})

	chainID := big.NewInt(a.Chainid) // Base, adapte selon ton fork
	signer := types.LatestSignerForChainID(chainID)
	signedTx, err := types.SignTx(tx, signer, a.Signer.key)
	if err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: sign: %w", err)
	}

	var txHash common.Hash
	if err := a.Client.CallCtx(ctx, eth.SendTx(signedTx).Returns(&txHash)); err != nil {
		return common.Hash{}, fmt.Errorf("SendSignedTx: send: %w", err)
	}
	fmt.Println("No error in liquidation")
	return txHash, nil
}

// ── LIQUIDATE ────────────────────────────────────────────────────────────────

func (a *AnvilInstance) LiquidateCall(ctx context.Context, args contract.LiquidateArgs, liquidatorAddr common.Address) (common.Hash, error) {
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
		return common.Hash{}, fmt.Errorf("LiquidateCall: encode: %w", err)
	}

	txHash, err := a.SendSignedTx(ctx, TxParams{
		To:       &liquidatorAddr,
		Calldata: calldata,
		Value:    big.NewInt(0),
	})
	if err != nil {
		return common.Hash{}, fmt.Errorf("LiquidateCall: %w", err)
	}
	if err := a.Mine(ctx); err != nil { // ← mine le bloc de la liquidation
		return common.Hash{}, fmt.Errorf("liquidate: mine: %w", err)
	}
	var receipt *types.Receipt
	if err := a.Client.CallCtx(ctx, eth.TxReceipt(txHash).Returns(&receipt)); err != nil {
		return common.Hash{}, fmt.Errorf("LiquidateCall: receipt: %w", err)
	}
	if receipt.Status == types.ReceiptStatusFailed {
		trace, err := a.CastRun(txHash)
		if err != nil {
			fmt.Printf("cast run failed: %v\n", err)
		} else {
			fmt.Println(trace)
		}
		return txHash, fmt.Errorf("LiquidateCall: REVERTED (hash: %s)", txHash.Hex())
	}
	fmt.Printf("[liquidate] tx: %s\n", txHash.Hex())
	return txHash, nil
}

func (a *AnvilInstance) DeployContract(ctx context.Context, bytecode []byte) (common.Address, error) {
	txHash, err := a.SendSignedTx(ctx, TxParams{
		To:       nil,
		Calldata: bytecode,
		Value:    big.NewInt(0),
	})
	if err != nil {
		return common.Address{}, fmt.Errorf("deploy: %w", err)
	}

	if err := a.Mine(ctx); err != nil { // ← mine le bloc de la liquidation
		return common.Address{}, fmt.Errorf("liquidate: mine: %w", err)
	}
	var receipt *types.Receipt
	for range 10 {
		err := a.Client.CallCtx(ctx, eth.TxReceipt(txHash).Returns(&receipt))
		if err == nil && receipt != nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if receipt.Status == types.ReceiptStatusFailed {
		return common.Address{}, fmt.Errorf("deploy: tx reverted (hash: %s)", txHash.Hex())
	}

	// ← vérification getCode
	var code []byte
	if err := a.Client.CallCtx(ctx, eth.Code(receipt.ContractAddress, nil).Returns(&code)); err != nil {
		return common.Address{}, fmt.Errorf("deploy: getCode failed: %w", err)
	}
	if len(code) == 0 {
		return common.Address{}, fmt.Errorf("deploy: no code at %s — deploy silently failed", receipt.ContractAddress.Hex())
	}
	fmt.Printf("[deploy] contrat déployé à %s (%d bytes)\n", receipt.ContractAddress.Hex(), len(code))

	return receipt.ContractAddress, nil
}

var funcBalanceOf = w3.MustNewFunc("balanceOf(address)", "uint256")

func (a *AnvilInstance) BalanceOf(ctx context.Context, token common.Address, account common.Address) (*big.Int, error) {
	var balance *big.Int
	fmt.Printf("  balanceOf token=%s account=%s\n", token.Hex(), account.Hex())

	if err := a.Client.CallCtx(ctx,
		eth.CallFunc(token, funcBalanceOf, account).Returns(&balance),
	); err != nil {
		return nil, fmt.Errorf("balanceOf: %w", err)
	}
	fmt.Printf("  balance result: %s\n", balance.String())
	return balance, nil
}

func (a *AnvilInstance) TraceTransaction(txHash common.Hash) (json.RawMessage, error) {
	rpcClient, err := rpc.Dial(a.RPC)
	if err != nil {
		return nil, err
	}
	defer rpcClient.Close()

	var result json.RawMessage
	err = rpcClient.Call(&result, "debug_traceTransaction", txHash.Hex(), map[string]string{
		"tracer": "callTracer",
	})
	if err != nil {
		return nil, fmt.Errorf("traceTransaction: %w", err)
	}
	return result, nil
}
