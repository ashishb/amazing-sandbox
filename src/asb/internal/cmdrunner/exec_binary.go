package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// resolveExecBinary resolves the binary to run for CmdTypeExec into an absolute
// path. Relative paths (e.g. ./bin/tool) are resolved against the working
// directory and bare names are looked up on the host's PATH.
// A macOS application bundle (e.g. "/Applications/Visual Studio Code.app") is
// resolved to the executable named by CFBundleExecutable in its Info.plist.
// The returned realPath has symlinks resolved, as sandboxes check the real file.
func resolveExecBinary(ctx context.Context, config Config) (absPath string, realPath string, err error) {
	return resolveExecBinaryWith(ctx, config, readBundleExecutable)
}

func resolveExecBinaryWith(ctx context.Context, config Config, bundleExecutable func(ctx context.Context, infoPlistPath string) (string, error),
) (absPath string, realPath string, err error) {
	if len(config.args) == 0 || config.args[0] == "" {
		return "", "", errors.New("no binary specified, usage: asb --mode=native exec <binary> [args...]")
	}

	binary := config.args[0]
	if strings.Contains(binary, "/") && !filepath.IsAbs(binary) {
		binary = filepath.Join(config.workingDir, binary)
	}

	if isAppBundle(binary) {
		binary, err = resolveAppBundleBinary(ctx, binary, bundleExecutable)
		if err != nil {
			return "", "", err
		}
	}

	absPath, err = exec.LookPath(binary)
	if err != nil {
		return "", "", fmt.Errorf("failed to find binary %q: %w", config.args[0], err)
	}

	if absPath, err = filepath.Abs(absPath); err != nil {
		return "", "", fmt.Errorf("failed to get absolute path of %q: %w", absPath, err)
	}

	realPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve symlinks of %q: %w", absPath, err)
	}
	return absPath, realPath, nil
}

// isAppBundle reports whether path is a macOS application bundle directory.
func isAppBundle(path string) bool {
	if !strings.HasSuffix(filepath.Clean(path), ".app") {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// resolveAppBundleBinary returns the path of the main executable inside a
// macOS application bundle, e.g.
// "/Applications/Visual Studio Code.app" -> ".../Contents/MacOS/Electron".
func resolveAppBundleBinary(ctx context.Context, bundlePath string,
	bundleExecutable func(ctx context.Context, infoPlistPath string) (string, error),
) (string, error) {
	bundlePath = filepath.Clean(bundlePath)
	infoPlistPath := filepath.Join(bundlePath, "Contents", "Info.plist")
	executable, err := bundleExecutable(ctx, infoPlistPath)
	if err != nil {
		return "", fmt.Errorf("failed to read CFBundleExecutable of %q: %w", bundlePath, err)
	}
	if executable == "" || strings.Contains(executable, "/") {
		return "", fmt.Errorf("invalid CFBundleExecutable %q in %q", executable, infoPlistPath)
	}
	return filepath.Join(bundlePath, "Contents", "MacOS", executable), nil
}

// readBundleExecutable reads CFBundleExecutable from an Info.plist using the
// macOS `defaults` tool.
func readBundleExecutable(ctx context.Context, infoPlistPath string) (string, error) {
	out, err := exec.CommandContext(ctx, "defaults", "read", infoPlistPath, "CFBundleExecutable").Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
