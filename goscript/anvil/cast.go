package anvil

import (
	"fmt"
	"os/exec"

	"github.com/ethereum/go-ethereum/common"
)

func (a *AnvilInstance) CastRun(txHash common.Hash) (string, error) {
	cmd := exec.Command("cast", "run",
		txHash.Hex(),
		"--rpc-url", a.RPC,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cast run: %w\n%s", err, string(out))
	}
	return string(out), nil
}
