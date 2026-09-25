package cmdrunner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExecArgs(t *testing.T) {
	t.Parallel()
	tests := []_TestData{
		{
			name:     "exec passes binary and args through unchanged",
			cmdType:  CmdTypeExec,
			args:     []string{"mytool", "--flag", "value"},
			wantArgs: []string{"mytool", "--flag", "value"},
		},
	}

	runTests(t, tests)
}

func TestResolveExecBinary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "mytool")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(dir, "mytool-link")
	if err := os.Symlink(binPath, linkPath); err != nil {
		t.Fatal(err)
	}
	wantRealPath, err := filepath.EvalSymlinks(binPath)
	if err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(CmdTypeExec, SetWorkingDir(dir), SetArgs([]string{"./mytool-link", "arg"}))
	absPath, realPath, err := resolveExecBinary(cfg)
	if err != nil {
		t.Fatalf("resolveExecBinary() error = %v", err)
	}
	if absPath != linkPath {
		t.Errorf("absPath = %q, want %q", absPath, linkPath)
	}
	if realPath != wantRealPath {
		t.Errorf("realPath = %q, want %q", realPath, wantRealPath)
	}

	cfg = NewConfig(CmdTypeExec, SetWorkingDir(dir), SetArgs([]string{"./does-not-exist"}))
	if _, _, err = resolveExecBinary(cfg); err == nil {
		t.Error("resolveExecBinary() expected error for missing binary")
	}
}

func TestExecRequiresNativeMode(t *testing.T) {
	t.Parallel()
	cfg := NewConfig(CmdTypeExec, SetArgs([]string{"true"}), SetExecMode(ExecModeDocker))
	if _, err := RunCmd(context.Background(), cfg); err == nil {
		t.Error("RunCmd() expected error for exec in docker mode")
	}
}
