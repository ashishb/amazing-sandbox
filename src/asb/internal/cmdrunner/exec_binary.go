package cmdrunner

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// resolveExecBinary resolves the binary to run for CmdTypeExec into an absolute
// path. Relative paths (e.g. ./bin/tool) are resolved against the working
// directory and bare names are looked up on the host's PATH.
// The returned realPath has symlinks resolved, as sandboxes check the real file.
func resolveExecBinary(config Config) (absPath string, realPath string, err error) {
	if len(config.args) == 0 || config.args[0] == "" {
		return "", "", errors.New("no binary specified, usage: asb --mode=native exec <binary> [args...]")
	}

	binary := config.args[0]
	if strings.Contains(binary, "/") && !filepath.IsAbs(binary) {
		binary = filepath.Join(config.workingDir, binary)
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
