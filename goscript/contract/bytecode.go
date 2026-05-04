package contract

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
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

func EncodedBytecodeWithConstructor(bytecode []byte, morphoBlueAddr common.Address) ([]byte, error) {
	// L'argument constructeur est une address
	addressType, _ := abi.NewType("address", "", nil)
	args := abi.Arguments{{Type: addressType}}

	encoded, err := args.Pack(morphoBlueAddr)
	if err != nil {
		return nil, fmt.Errorf("constructor encode: %w", err)
	}

	// bytecode + args ABI-encodés
	return append(bytecode, encoded...), nil
}
