package contract

import (
	"encoding/json"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

type ForgeArtifact struct {
	Bytecode struct {
		Object string `json:"object"`
	} `json:"bytecode"`
}

func LoadBytecode(artifactPath string) ([]byte, error) {
	raw, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, err
	}
	var artifact ForgeArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return nil, err
	}
	return common.FromHex(artifact.Bytecode.Object), nil
}
