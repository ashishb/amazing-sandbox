//go:build !darwin && !linux

package cmdrunner

import (
	"context"
	"fmt"
	"runtime"
)

// runCmdInNative is a placeholder for platforms that do not have a native
// sandboxing backend. Native mode is only implemented for macOS (sandbox-exec)
// and Linux (bubblewrap); everywhere else it returns an error so callers can
// fall back to Docker.
func runCmdInNative(_ context.Context, _ Config) (*ShellResult, error) {
	return nil, fmt.Errorf("native mode is not supported on %s; use --mode=docker instead", runtime.GOOS)
}
