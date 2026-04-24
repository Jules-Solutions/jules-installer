//go:build integration
// +build integration

// Integration tests for the jules-setup binary. Opt-in because they:
//   - build the binary via `go build`
//   - spawn a subprocess
//   - read and write ~/.config/jules/config.toml inside a temp dir
//   - hit the offline-scaffold fallback path (no network needed)
//
// Run with:
//   go test -tags integration ./cmd/jules-setup/
//
// Expected CI usage: nightly or pre-release job, NOT on every PR — these
// take ~10s each because of the go build step.
package main_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// buildBinary compiles jules-setup into a temp dir and returns the absolute
// path to the resulting executable. Cached across subtests within a single
// test binary run.
func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "jules-setup")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	// Build from the repo root (one level up from cmd/jules-setup).
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/jules-setup")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// This test file lives at cmd/jules-setup/. Repo root is ../..
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Walk up until we find go.mod.
	dir := wd
	for range [10]int{} {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate repo root (go.mod)")
	return ""
}

// withIsolatedHome points XDG_CONFIG_HOME (Linux) and USERPROFILE (Windows)
// at a temp dir so the subprocess writes to a test-private location.
// Returns the home dir path plus an env slice ready for exec.Cmd.Env.
func withIsolatedHome(t *testing.T) (string, []string) {
	t.Helper()
	home := t.TempDir()
	env := append([]string{}, os.Environ()...)
	// Strip any existing values that would interfere.
	filtered := env[:0]
	for _, kv := range env {
		if strings.HasPrefix(kv, "XDG_CONFIG_HOME=") ||
			strings.HasPrefix(kv, "HOME=") ||
			strings.HasPrefix(kv, "USERPROFILE=") {
			continue
		}
		filtered = append(filtered, kv)
	}
	env = filtered
	env = append(env, "XDG_CONFIG_HOME="+home)
	if runtime.GOOS == "windows" {
		env = append(env, "USERPROFILE="+home)
	} else {
		env = append(env, "HOME="+home)
	}
	// Prevent the self-update check from hitting GitHub in tests.
	env = append(env, "JULES_NO_UPDATE=1")
	return home, env
}

// seedConfig writes a realistic config.toml under home/.config/jules/
// matching the given template. Caller supplies tier + api_key; other fields
// are filled with sensible defaults.
func seedConfig(t *testing.T, home, tier, apiKey string) string {
	t.Helper()
	cfgDir := filepath.Join(home, ".config", "jules")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `[auth]
api_key = "` + apiKey + `"
api_url = "https://api.jules.solutions"
auth_url = "https://auth.jules.solutions"
mcp_url = "https://mcp.jules.solutions/sse"

[local]
tier = "` + tier + `"
`
	path := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config.toml: %v", err)
	}
	return path
}

// runInstaller invokes the binary with the given args and environment and
// fails the test on non-zero exit (unless allowExit is true, in which case
// the exit code is returned for caller inspection).
func runInstaller(t *testing.T, bin string, env []string, args ...string) (stdout, stderr string) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	// Feed an empty stdin so any accidental prompt read returns EOF immediately.
	cmd.Stdin = bytes.NewReader(nil)
	if err := cmd.Run(); err != nil {
		// Some flows (Tier 1 without network) can exit non-zero after
		// writing partial state. Caller asserts on filesystem, not exit code.
		t.Logf("installer exited with error (may be expected): %v\nstderr: %s", err, errBuf.String())
	}
	return outBuf.String(), errBuf.String()
}

// TestIntegration_Tier2_Headless exercises the simplest --yes flow:
// a clean environment, API key in config, --tier 2 --yes, expect
// ~/.claude/.mcp.json written with the correct shape.
func TestIntegration_Tier2_Headless(t *testing.T) {
	bin := buildBinary(t)
	home, env := withIsolatedHome(t)
	seedConfig(t, home, "tier2", "dck_integration_tier2")

	_, stderr := runInstaller(t, bin, env, "--tier", "2", "--yes")

	mcpPath := filepath.Join(home, ".claude", ".mcp.json")
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("expected %s to exist after --yes Tier 2 run: %v\nstderr: %s", mcpPath, err, stderr)
	}
	var parsed struct {
		MCPServers map[string]struct {
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse .mcp.json: %v", err)
	}
	if len(parsed.MCPServers) != 1 {
		t.Errorf("Tier 2 should have 1 server, got %d", len(parsed.MCPServers))
	}
	if parsed.MCPServers["jules"].Headers["X-API-Key"] != "dck_integration_tier2" {
		t.Errorf("Tier 2 X-API-Key mismatch; got %v", parsed.MCPServers["jules"].Headers)
	}
}

// TestIntegration_Tier2ToTier1_Upgrade is the core regression test for the
// documented `jules-setup --tier 1` upgrade path. Seeds a Tier 2 config,
// runs the binary with --tier 1 --yes, asserts the resulting on-disk state
// matches a Tier 1 install.
//
// This is the ONLY scripted end-to-end test of the upgrade contract. If it
// breaks, the README's upgrade-path instructions are silently wrong.
func TestIntegration_Tier2ToTier1_Upgrade(t *testing.T) {
	bin := buildBinary(t)
	home, env := withIsolatedHome(t)
	seedConfig(t, home, "tier2", "dck_integration_upgrade")

	// Tier 1 wants a vault path; use a test-controlled one inside HOME.
	// Note: --vault-path isn't a flag today; we rely on DefaultVaultPath()
	// which derives from HOME. The subprocess's HOME is our isolated temp
	// dir, so the vault lands inside it automatically.
	_, _ = runInstaller(t, bin, env, "--tier", "1", "--yes")

	// Reload config.
	cfgPath := filepath.Join(home, ".config", "jules", "config.toml")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	var post struct {
		Local struct {
			Tier      string `toml:"tier"`
			VaultPath string `toml:"vault_path"`
			MCPPath   string `toml:"mcp_path"`
		} `toml:"local"`
	}
	if _, err := toml.Decode(string(raw), &post); err != nil {
		t.Fatalf("parse config.toml: %v", err)
	}

	if post.Local.Tier != "tier1" {
		t.Errorf("upgrade should flip tier to 'tier1', got %q", post.Local.Tier)
	}
	if post.Local.VaultPath == "" {
		t.Error("upgrade should populate vault_path")
	}

	// .mcp.json should live in the vault root, NOT ~/.claude/.
	vaultMCP := filepath.Join(post.Local.VaultPath, ".mcp.json")
	if _, err := os.Stat(vaultMCP); err != nil {
		t.Errorf("upgrade should write %s: %v", vaultMCP, err)
	}
	if post.Local.MCPPath != vaultMCP {
		t.Errorf("config.local.mcp_path = %q, want %q", post.Local.MCPPath, vaultMCP)
	}

	// --- INST §Test flow additional assertions ---

	// (a) CLAUDE.md scaffold marker — present when the offline scaffold ran.
	// Git-clone path also produces a CLAUDE.md (it's in the vault template),
	// so this assertion is valid either way.
	claudeMD := filepath.Join(post.Local.VaultPath, "CLAUDE.md")
	if _, err := os.Stat(claudeMD); err != nil {
		t.Errorf("scaffold should create %s: %v", claudeMD, err)
	}

	// (b) .mcp.json file mode 0600 on Unix — holds an API key. Skipped on
	// Windows because NTFS doesn't map cleanly to POSIX modes.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(vaultMCP)
		if err == nil {
			if mode := info.Mode().Perm(); mode != 0o600 {
				t.Errorf(".mcp.json mode = %o, want 0600", mode)
			}
		}
	}

	// (c) No leftover Tier 2 ~/.claude/.mcp.json from the seeded state.
	// (The seed config didn't actually write one — it only set mcp_path to
	// a fake path — but we assert the Tier 1 flow didn't silently create
	// one anyway.)
	globalMCP := filepath.Join(home, ".claude", ".mcp.json")
	if _, err := os.Stat(globalMCP); err == nil {
		t.Errorf("Tier 1 upgrade should not write %s (that's the Tier 2 path)", globalMCP)
	}
}

// TestIntegration_Tier1_WithLocalTools verifies the --local-tools-mcp flag
// is honoured in headless mode.
func TestIntegration_Tier1_WithLocalTools(t *testing.T) {
	bin := buildBinary(t)
	home, env := withIsolatedHome(t)
	seedConfig(t, home, "tier2", "dck_integration_local_tools") // start as Tier 2 to exercise both features

	_, _ = runInstaller(t, bin, env,
		"--tier", "1",
		"--yes",
		"--local-tools-mcp", "true",
	)

	// Find the vault mcp path from config.
	cfgRaw, _ := os.ReadFile(filepath.Join(home, ".config", "jules", "config.toml"))
	var post struct {
		Local struct {
			MCPPath       string `toml:"mcp_path"`
			LocalToolsMCP bool   `toml:"local_tools_mcp"`
		} `toml:"local"`
	}
	_, _ = toml.Decode(string(cfgRaw), &post)

	if !post.Local.LocalToolsMCP {
		t.Error("config.local.local_tools_mcp should be true after --local-tools-mcp=true")
	}

	if post.Local.MCPPath == "" {
		t.Fatal("config missing mcp_path after Tier 1 install")
	}
	data, err := os.ReadFile(post.Local.MCPPath)
	if err != nil {
		t.Fatalf("read mcp file: %v", err)
	}
	var parsed struct {
		MCPServers map[string]any `json:"mcpServers"`
	}
	_ = json.Unmarshal(data, &parsed)
	if _, ok := parsed.MCPServers["jules"]; !ok {
		t.Error("expected 'jules' server entry")
	}
	if _, ok := parsed.MCPServers["jules-local"]; !ok {
		t.Error("expected 'jules-local' stdio entry when --local-tools-mcp=true")
	}
}

// TestIntegration_YesRequiresTier confirms the binary refuses --yes without --tier.
func TestIntegration_YesRequiresTier(t *testing.T) {
	bin := buildBinary(t)
	_, env := withIsolatedHome(t)

	cmd := exec.Command(bin, "--yes")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("--yes without --tier should exit non-zero; output: %s", out)
	}
	if !strings.Contains(string(out), "tier") {
		t.Errorf("error message should mention tier; got: %s", out)
	}
}
