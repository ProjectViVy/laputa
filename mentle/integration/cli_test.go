package integration

import (
	"os"
	"os/exec"
	"testing"
)

func TestCLIInit(t *testing.T) {
	tmp := t.TempDir()

	cmd := exec.Command("go", "run", ".", "init", tmp)
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Output: %s", string(output))
		t.Fatalf("init command failed: %v", err)
	}

	if info, err := os.Stat(tmp); err != nil || !info.IsDir() {
		t.Errorf("expected palace directory to be created: %v", err)
	}
}

func TestCLIStatus(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "status")
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Output: %s", string(output))
		t.Fatalf("status command failed: %v", err)
	}

	outputStr := string(output)
	if outputStr == "" {
		t.Error("status command produced no output")
	}

	if !containsSubstring(outputStr, "Palace:") {
		t.Error("status output should contain 'Palace:'")
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCLIHelp(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--help")
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	outputStr := string(output)
	expectedCommands := []string{"init", "mine", "search", "status", "wake-up"}

	for _, cmdName := range expectedCommands {
		if !containsSubstring(outputStr, cmdName) {
			t.Errorf("help should mention command: %s", cmdName)
		}
	}
}

func TestCLIInvalidCommand(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "nonexistent-command")
	cmd.Dir = ".."
	output, _ := cmd.CombinedOutput()

	outputStr := string(output)
	if !containsSubstring(outputStr, "unknown") && !containsSubstring(outputStr, "Available Commands") {
		t.Errorf("invalid command should show error or available commands, got: %s", outputStr)
	}
}
