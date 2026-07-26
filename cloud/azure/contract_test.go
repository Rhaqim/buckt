package azure

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

type contractBackend interface {
	Name() string
	Put(ctx context.Context, path string, data []byte) error
	Get(ctx context.Context, path string) ([]byte, error)
	Exists(ctx context.Context, path string) (bool, error)
	List(ctx context.Context, prefix string) ([]string, error)
	Move(ctx context.Context, oldPath, newPath string) error
	Delete(ctx context.Context, path string) error
	DeleteFolder(ctx context.Context, prefix string) error
}

func runBackendContract(t *testing.T, be contractBackend) {
	t.Helper()
	ctx := context.Background()
	must := func(err error, what string) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}

	dir := "contract"
	_ = be.DeleteFolder(ctx, dir)
	a := dir + "/a.txt"
	b := dir + "/b.txt"
	data := []byte("hello " + be.Name())

	must(be.Put(ctx, a, data), "Put")

	got, err := be.Get(ctx, a)
	must(err, "Get")
	if !bytes.Equal(got, data) {
		t.Fatalf("Get mismatch: got %q want %q", got, data)
	}

	ok, err := be.Exists(ctx, a)
	must(err, "Exists(existing)")
	if !ok {
		t.Fatal("Exists returned false for an existing object")
	}
	ok, err = be.Exists(ctx, dir+"/missing.txt")
	must(err, "Exists(missing)")
	if ok {
		t.Fatal("Exists returned true for a missing object")
	}

	keys, err := be.List(ctx, dir+"/")
	must(err, "List")
	if !anyHasSuffix(keys, "a.txt") {
		t.Fatalf("List did not include a.txt: %v", keys)
	}

	must(be.Move(ctx, a, b), "Move")
	got, err = be.Get(ctx, b)
	must(err, "Get(after move)")
	if !bytes.Equal(got, data) {
		t.Fatal("moved object content mismatch")
	}
	if ok, _ := be.Exists(ctx, a); ok {
		t.Error("source still exists after Move")
	}

	must(be.Delete(ctx, b), "Delete")
	if ok, _ := be.Exists(ctx, b); ok {
		t.Error("object still exists after Delete")
	}

	must(be.Put(ctx, dir+"/c.txt", []byte("x")), "Put(c)")
	must(be.DeleteFolder(ctx, dir), "DeleteFolder")
	keys, err = be.List(ctx, dir+"/")
	must(err, "List(after DeleteFolder)")
	if len(keys) != 0 {
		t.Fatalf("expected no objects after DeleteFolder, got: %v", keys)
	}
}

func anyHasSuffix(keys []string, suffix string) bool {
	for _, k := range keys {
		if strings.HasSuffix(k, suffix) {
			return true
		}
	}
	return false
}

// Azurite's well-known development account credentials.
const (
	azuriteAccount = "devstoreaccount1"
	azuriteKey     = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
)

func TestAzureBackend_Contract_Azurite(t *testing.T) {
	endpoint := os.Getenv("BUCKT_AZURITE_ENDPOINT") // e.g. http://127.0.0.1:10000/devstoreaccount1
	if endpoint == "" {
		t.Skip("BUCKT_AZURITE_ENDPOINT not set; skipping Azure emulator contract test")
	}
	containerName := os.Getenv("BUCKT_AZURITE_CONTAINER")
	if containerName == "" {
		containerName = "buckt-test"
	}

	be, err := NewBackend(Config{
		AccountName: azuriteAccount,
		AccountKey:  azuriteKey,
		Container:   containerName,
		Endpoint:    endpoint,
	})
	if err != nil {
		t.Fatalf("NewBackend: %v", err)
	}
	// Create the container (idempotent) directly via the backend's client.
	_, _ = be.client.Create(context.Background(), nil)

	runBackendContract(t, be)
}
