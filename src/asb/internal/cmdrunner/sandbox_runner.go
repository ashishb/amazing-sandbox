package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
)

// runSandboxCmd runs the command using macOS sandbox-exec for sandboxing.
// It fails on non-macOS systems.
func runSandboxCmd(ctx context.Context, config Config) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("--mode=sandbox is only supported on macOS (darwin), current OS: %s", runtime.GOOS)
	}

	if len(config.args) == 0 {
		return fmt.Errorf("no command specified for sandbox mode")
	}

	profile := generateSandboxProfile(config)
	log.Debug().
		Str("profile", profile).
		Msg("Generated sandbox-exec profile")

	// Build: sandbox-exec -p '<profile>' <cmd> [args...]
	sandboxArgs := []string{"-p", profile}
	sandboxArgs = append(sandboxArgs, config.args...)

	log.Debug().
		Strs("sandboxArgs", sandboxArgs).
		Msg("Running command with sandbox-exec")

	//nolint:gosec  // User is deliberately executing a command
	cmdCtx := exec.CommandContext(ctx, "sandbox-exec", sandboxArgs...)
	if isInteractiveTerminal() {
		cmdCtx.Stdin = os.Stdin
		cmdCtx.Stdout = os.Stdout
		cmdCtx.Stderr = os.Stderr
	}

	err := cmdCtx.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}

	if err != nil {
		return fmt.Errorf("failed to run sandbox-exec: %w", err)
	}

	log.Debug().
		Strs("sandboxArgs", sandboxArgs).
		Msg("sandbox-exec ran successfully")
	return nil
}

// generateSandboxProfile produces a macOS SBPL sandbox profile based on config.
func generateSandboxProfile(config Config) string {
	var sb strings.Builder

	sb.WriteString("(version 1)\n")
	sb.WriteString("(deny default)\n")

	// Allow basic process and IPC operations
	sb.WriteString("(allow process-exec*)\n")
	sb.WriteString("(allow process-fork)\n")
	sb.WriteString("(allow signal)\n")
	sb.WriteString("(allow sysctl-read)\n")
	sb.WriteString("(allow mach-lookup)\n")
	sb.WriteString("(allow mach-register)\n")
	sb.WriteString("(allow ipc-posix*)\n")

	// Allow reading essential system paths
	for _, p := range []string{
		"/usr",
		"/bin",
		"/sbin",
		"/System",
		"/Library",
		"/private/etc",
		"/private/var/db",
	} {
		fmt.Fprintf(&sb, "(allow file-read* (subpath %q))\n", p)
	}

	// Allow reading file metadata everywhere (needed for many tools)
	sb.WriteString("(allow file-read-metadata)\n")

	// Allow read-write access to temporary directories
	for _, p := range []string{"/tmp", "/private/tmp", "/var/folders"} {
		fmt.Fprintf(&sb, "(allow file-read* file-write* (subpath %q))\n", p)
	}

	// Allow /dev access
	sb.WriteString("(allow file-read* file-write* (subpath \"/dev\"))\n")

	// Allow reading home directory (tool configs, caches, etc.)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		fmt.Fprintf(&sb, "(allow file-read* (subpath %q))\n", homeDir)
	}

	// Working directory access
	if config.mountWorkingDirRW {
		fmt.Fprintf(&sb, "(allow file-read* file-write* (subpath %q))\n", config.workingDir)
	} else if config.mountWorkingDirRO {
		fmt.Fprintf(&sb, "(allow file-read* (subpath %q))\n", config.workingDir)
	}

	// Referenced files/directories
	for _, dir := range config.getReferencedFiles() {
		if config.mountReferencedDirRW {
			fmt.Fprintf(&sb, "(allow file-read* file-write* (subpath %q))\n", dir)
		} else if config.mountReferencedDirRO {
			fmt.Fprintf(&sb, "(allow file-read* (subpath %q))\n", dir)
		}
	}

	// Extra read-only directories
	for _, dir := range config.extraMountRODirs {
		fmt.Fprintf(&sb, "(allow file-read* (subpath %q))\n", dir)
	}

	// Network access
	if config.networkType != NetworkNone {
		sb.WriteString("(allow network*)\n")
	}

	return sb.String()
}
