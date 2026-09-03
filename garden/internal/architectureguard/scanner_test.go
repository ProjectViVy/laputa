package architectureguard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFindsRuntimeViolationsAndSkipsHistoricalDocumentsAndTests(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("garden/main.go", `package main
import "github.com/dashimaki/laputa/governance"
const route = "/v2/cognitive/world"
`)
	write("garden/main_test.go", `package main
const old = "/v2/governance/projection"
`)
	write("docs/archive/old.md", "github.com/dashimaki/laputa/governance /v2/cognitive/world")

	findings, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings=%+v, want two runtime findings", findings)
	}
	if findings[0].Path != "garden/main.go" || findings[0].Rule != "legacy-laputa-governance" {
		t.Fatalf("findings=%+v", findings)
	}
	if findings[1].Path != "garden/main.go" || findings[1].Rule != "legacy-governance-routes" {
		t.Fatalf("findings=%+v", findings)
	}
}

func TestScanFindsMentleAuthorityDependency(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "mentle", "facade.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package facade\nconst authority = \"WORLD.MD\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Rule != "mentle-authority-boundary" {
		t.Fatalf("findings=%+v", findings)
	}
}

func TestExitCodeOnlyBlocksInEnforceMode(t *testing.T) {
	if got := ExitCode(ModeReport, 4); got != 0 {
		t.Fatalf("report exit=%d", got)
	}
	if got := ExitCode(ModeEnforce, 0); got != 0 {
		t.Fatalf("clean enforce exit=%d", got)
	}
	if got := ExitCode(ModeEnforce, 1); got != 1 {
		t.Fatalf("dirty enforce exit=%d", got)
	}
}
