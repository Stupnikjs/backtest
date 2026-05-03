package anvil

import (
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"time"

	"github.com/lmittmann/w3"
	"github.com/lmittmann/w3/module/eth"
)

// ─── Anvil ────────────────────────────────────────────────────

type AnvilInstance struct {
	Port    int
	Cmd     *exec.Cmd
	RPC     string
	Client  *w3.Client
	Signer  Signer
	Chainid int64
}

func startAnvil(forkURL string, blockNumber uint64, port int) (*AnvilInstance, error) {
	cmd := exec.Command("anvil",
		"--fork-url", forkURL,
		"--fork-block-number", fmt.Sprintf("%d", blockNumber), // juste avant la liquidation
		"--port", fmt.Sprintf("%d", port),
		"--no-mining", // mining manuel = contrôle total
		"--silent",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("anvil start failed: %w", err)
	}

	// Attendre qu'Anvil soit prêt
	rpc := fmt.Sprintf("http://localhost:%d", port)
	client, err := waitForAnvil(rpc, 10*time.Second)
	if err != nil {
		cmd.Process.Kill()
		return nil, err
	}

	return &AnvilInstance{
		Port:   port,
		Cmd:    cmd,
		RPC:    rpc,
		Client: client,
	}, nil
}

func waitForAnvil(rpc string, timeout time.Duration) (*w3.Client, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		client, err := w3.Dial(rpc)
		if err == nil {
			var blockNum *big.Int
			if err := client.Call(eth.BlockNumber().Returns(&blockNum)); err == nil {
				return client, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil, fmt.Errorf("anvil timeout after %s", timeout)
}

func (a *AnvilInstance) Stop() {
	a.Client.Close()
	a.Cmd.Process.Kill()
}
