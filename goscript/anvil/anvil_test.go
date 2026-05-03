package anvil

import (
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	// Remonte jusqu'à trouver le .env
	for _, path := range []string{".env", "../.env", "../../.env"} {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}
	os.Exit(m.Run())
}

func TestStartAndStop(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("getFreePort: %v", err)
	}

	instance, err := StartAnvil(os.Getenv("ARB_RPC"), 18000000, port)
	if err != nil {
		t.Fatalf("startAnvil: %v", err)
	}
	defer instance.Stop()

	if instance.Client == nil {
		t.Fatal("client is nil")
	}
	if instance.Port != port {
		t.Errorf("expected port %d, got %d", port, instance.Port)
	}
}

func TestWaitForAnvilTimeout(t *testing.T) {
	_, err := waitForAnvil("http://localhost:19999", 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestMultipleInstances(t *testing.T) {
	const n = 3
	instances := make([]*AnvilInstance, 0, n)

	for i := 0; i < n; i++ {
		port, err := getFreePort()
		if err != nil {
			t.Fatalf("getFreePort instance %d: %v", i, err)
		}

		inst, err := StartAnvil(os.Getenv("ARB_RPC"), 18000000, port)
		if err != nil {
			t.Fatalf("startAnvil instance %d: %v", i, err)
		}
		instances = append(instances, inst)
	}

	defer func() {
		for _, inst := range instances {
			inst.Stop()
		}
	}()

	ports := make(map[int]bool)
	for _, inst := range instances {
		if ports[inst.Port] {
			t.Errorf("port collision: %d utilisé deux fois", inst.Port)
		}
		ports[inst.Port] = true
	}
}
