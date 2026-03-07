package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultAdminUIDir(t *testing.T) {
	t.Setenv("XBOARD_UI_ADMIN_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.UI.Admin.Dir != "web/user-vite/dist" {
		t.Fatalf("expected ui.admin.dir default to %q, got %q", "web/user-vite/dist", cfg.UI.Admin.Dir)
	}
}

func TestLoad_GrpcAddressBackwardCompatibility(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yml")

	t.Setenv("XBOARD_GRPC_ADDR", "")
	t.Setenv("XBOARD_GRPC_ADDRESS", "")

	content := []byte("grpc:\n  address: \"127.0.0.1:19100\"\n")
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.GRPC.Addr != "127.0.0.1:19100" {
		t.Fatalf("expected grpc.addr fallback from grpc.address, got %q", cfg.GRPC.Addr)
	}
}
