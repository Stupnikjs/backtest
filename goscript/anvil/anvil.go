package anvil

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

const (
	defaultTimeout = 10 * time.Second
	defaultGas     = 3_000_000
	pollInterval   = 200 * time.Millisecond
	anvilBlockTime = "1"
)

// ─── Anvil ────────────────────────────────────────────────────

type AnvilInstance struct {
	Port      int
	Cmd       *exec.Cmd
	RPC       string
	Client    *w3.Client
	RpcClient *rpc.Client
	Signer    Signer
	Chainid   int64
}

func StartAnvil(rpcUrl string, blockNumber uint64, port int, chainid int) (*AnvilInstance, error) {
	cmd := anvilCmd(rpcUrl, blockNumber+2, port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("anvil start: %w", err)
	}

	forkUrl := fmt.Sprintf("http://localhost:%d", port)
	client, err := waitForAnvil(forkUrl, defaultTimeout)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("anvil start (block=%d port=%d): %w", blockNumber, port, err)
	}
	rpcClient, err := rpc.Dial(forkUrl)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("rpc dial: %w", err)
	}

	signer, err := NewAnvilSigner(int64(chainid))
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("anvil signer: %w", err)
	}

	return &AnvilInstance{
		Port:      port,
		Cmd:       cmd,
		RPC:       forkUrl,
		Client:    client,
		RpcClient: rpcClient,
		Signer:    *signer,
		Chainid:   int64(chainid),
	}, nil
}

func waitForAnvil(forkurl string, timeout time.Duration) (*w3.Client, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if client, err := tryDial(forkurl); err == nil {
			return client, nil
		}
		time.Sleep(pollInterval)
	}
	return nil, fmt.Errorf("timeout waiting for anvil on %s", forkurl)
}

func tryDial(forkurl string) (*w3.Client, error) {
	client, err := w3.Dial(forkurl)
	if err != nil {
		return nil, err
	}
	var blockNum *big.Int
	if err := client.Call(eth.BlockNumber().Returns(&blockNum)); err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}

func anvilCmd(rpc string, blockNumber uint64, port int) *exec.Cmd {
	return exec.Command("anvil",
		"--fork-url", rpc,
		"--fork-block-number", fmt.Sprintf("%d", blockNumber),
		"--port", fmt.Sprintf("%d", port),
		"--no-mining", // ← plus de block-time
		"--silent",
	)
}

func (a *AnvilInstance) Mine(ctx context.Context) error {
	rpcClient, err := rpc.DialContext(ctx, a.RPC)
	if err != nil {
		return err
	}
	defer rpcClient.Close()
	return rpcClient.CallContext(ctx, nil, "anvil_mine", "0x1")
}

func (a *AnvilInstance) Stop() {
	if a.Client != nil {
		a.Client.Close()
	}
	if a.RpcClient != nil {
		a.RpcClient.Close()
	}
	if a.Cmd != nil && a.Cmd.Process != nil {
		a.Cmd.Process.Kill()
	}
}
