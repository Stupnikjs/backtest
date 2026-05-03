package contract

import (
	"os/exec"
	"testing"
)

func TestLoadBytecode(t *testing.T) {
	// 1. forge build
	cmd := exec.Command("forge", "build")
	cmd.Dir = "../../" // racine du projet Foundry
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("forge build failed: %v\n%s", err, out)
	}
	t.Log("forge build OK")

	// 2. charger le bytecode
	bytecode, err := LoadBytecode("../../out/Liquidator.sol/Liquidator.json")
	if err != nil {
		t.Fatalf("LoadBytecode: %v", err)
	}
	if len(bytecode) == 0 {
		t.Fatal("bytecode vide")
	}

	t.Logf("bytecode chargé: %d bytes", len(bytecode))
}
