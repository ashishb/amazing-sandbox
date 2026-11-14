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
	absPath, realPath, err := resolveExecBinary(t.Context(), cfg)
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
	if _, _, err = resolveExecBinary(t.Context(), cfg); err == nil {
		t.Error("resolveExecBinary() expected error for missing binary")
	}
}

func TestResolveExecBinaryAppBundle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bundle := filepath.Join(dir, "My App.app")
	macOSDir := filepath.Join(bundle, "Contents", "MacOS")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(macOSDir, "Electron")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	wantInfoPlist := filepath.Join(bundle, "Contents", "Info.plist")
	fakeReader := func(_ context.Context, infoPlistPath string) (string, error) {
		if infoPlistPath != wantInfoPlist {
			t.Errorf("Info.plist path = %q, want %q", infoPlistPath, wantInfoPlist)
		}
		return "Electron", nil
	}

	for _, arg := range []string{bundle, bundle + "/", "./My App.app"} {
		cfg := NewConfig(CmdTypeExec, SetWorkingDir(dir), SetArgs([]string{arg, "--no-sandbox"}))
		absPath, _, err := resolveExecBinaryWith(t.Context(), cfg, fakeReader)
		if err != nil {
			t.Fatalf("resolveExecBinaryWith(%q) error = %v", arg, err)
		}
		if absPath != binPath {
			t.Errorf("resolveExecBinaryWith(%q) absPath = %q, want %q", arg, absPath, binPath)
		}
	}

	badReader := func(context.Context, string) (string, error) { return "../evil", nil }
	cfg := NewConfig(CmdTypeExec, SetWorkingDir(dir), SetArgs([]string{bundle}))
	if _, _, err := resolveExecBinaryWith(t.Context(), cfg, badReader); err == nil {
		t.Error("resolveExecBinaryWith() expected error for invalid CFBundleExecutable")
	}
}

func TestExecRequiresNativeMode(t *testing.T) {
	t.Parallel()
	cfg := NewConfig(CmdTypeExec, SetArgs([]string{"true"}), SetExecMode(ExecModeDocker))
	if _, err := RunCmd(context.Background(), cfg); err == nil {
		t.Error("RunCmd() expected error for exec in docker mode")
	}
}
