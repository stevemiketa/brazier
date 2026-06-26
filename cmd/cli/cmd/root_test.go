package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brazier/brazier/cmd/cli/cmd"
)

func execute(args ...string) (string, error) {
	root := cmd.Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestRootHelp(t *testing.T) {
	out, err := execute("--help")
	if err != nil {
		t.Fatalf("help: %v", err)
	}
	for _, want := range []string{"brazier", "run", "logs", "status", "trigger", "agent"} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q:\n%s", want, out)
		}
	}
}

func TestLogsRequiresArg(t *testing.T) {
	_, err := execute("logs")
	if err == nil {
		t.Error("expected error when run-id missing")
	}
}

func TestStatusRequiresArg(t *testing.T) {
	_, err := execute("status")
	if err == nil {
		t.Error("expected error when run-id missing")
	}
}

func TestTriggerRequiresArg(t *testing.T) {
	_, err := execute("trigger")
	if err == nil {
		t.Error("expected error when project missing")
	}
}

func TestAgentStartRequiresName(t *testing.T) {
	// --name is empty by default; RunE should return an error before dialing.
	_, err := execute("agent", "start", "--master", "localhost:19999")
	if err == nil {
		t.Error("expected error when --name is missing")
	}
}

func TestRunHelp(t *testing.T) {
	out, err := execute("run", "--help")
	if err != nil {
		t.Fatalf("run help: %v", err)
	}
	if !strings.Contains(out, "--dir") {
		t.Errorf("run help missing --dir flag:\n%s", out)
	}
}
