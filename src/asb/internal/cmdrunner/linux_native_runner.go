//go:build linux

package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// runCmdInNative runs the command on Linux using bubblewrap (bwrap) as the
// sandboxing mechanism. This mirrors the sandbox-exec based implementation used
// on macOS: the whole host filesystem is exposed read-only and only a curated
// set of paths (the working directory, referenced files and package-manager
// caches) are made writable. Network access is toggled via a network namespace.
//
// Ref: https://github.com/containers/bubblewrap
func runCmdInNative(ctx context.Context, config Config) (*ShellResult, error) {
	if _, err := exec.LookPath("bwrap"); err != nil {
		return nil, fmt.Errorf(
			"native mode on Linux requires bubblewrap (bwrap) to be installed and on PATH: %w", err)
	}

	if err := ensureBubblewrapCanSandbox(ctx); err != nil {
		return nil, err
	}

	// config.args is already normalized by SetArgs (e.g. ["node", "index.js"]).
	cmdArgs := config.args
	if len(cmdArgs) == 0 {
		return nil, errors.New("no command to run in native mode")
	}

	bwrapArgs := []string{
		"bwrap",
		// Expose the entire host filesystem read-only, then overlay writable paths
		// on top of it. This matches the macOS policy of "allow reads everywhere,
		// restrict writes".
		"--ro-bind", "/bin", "/bin",
		"--ro-bind", "/etc", "/etc",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
		"--ro-bind", "/sbin", "/sbin",
		"--ro-bind", "/opt", "/opt",
		"--ro-bind", "/run", "/run",
		"--ro-bind", "/snap", "/snap",
		"--ro-bind", "/sys", "/sys",
		"--ro-bind", "/usr/", "/usr",
		// Fresh pseudo-filesystems. These come after the root bind so they take
		// precedence, and give the sandbox a clean /dev (null, zero, random,
		// urandom, tty, ...), /proc and writable temporary directories.
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
		"--tmpfs", "/var/tmp",
		// Namespace isolation. Use the "-try" variant for the user namespace: when
		// bwrap is not setuid it needs a user namespace to gain privileges, but on
		// hosts that grant privileges another way (e.g. real root with CAP_SYS_ADMIN)
		// forcing --unshare-user would fail unnecessarily.
		"--unshare-user-try",
		"--unshare-ipc",
		"--unshare-pid",
		"--unshare-uts",
		"--unshare-cgroup-try",
		// Ensure the sandboxed process is reaped if the parent dies.
		"--die-with-parent",
	}

	if config.networkType == NetworkNone {
		// Drop the network namespace entirely to deny all network access.
		bwrapArgs = append(bwrapArgs, "--unshare-net")
	}
	// Otherwise the host network namespace is retained, which also gives the
	// sandboxed process access to /etc/resolv.conf (exposed via the read-only
	// root bind) for DNS resolution.

	// Writable package-manager caches so repeated runs are fast. The parent of
	// each of these is read-only inside the sandbox, so the directory has to exist
	// on the host before it can be bind-mounted; create it if necessary.
	rwPathsToMount := []string{
		os.ExpandEnv("$HOME/go"), // Go module and build cache
	}

	rwPathsToMount = append(rwPathsToMount, getCachesForPackageManagers()...)
	for _, env := range []string{"UV_CACHE_DIR", "ZIG_GLOBAL_CACHE_DIR", "ZIG_LOCAL_CACHE_DIR"} {
		if os.Getenv(env) != "" {
			log.Debug().
				Str(env, os.Getenv(env)).
				Msgf("Adding cache dir %s to read/write path mounts", env)
			rwPathsToMount = append(rwPathsToMount, os.ExpandEnv(os.Getenv(env)))
		}
	}

	for _, path := range rwPathsToMount {
		if err := os.MkdirAll(path, 0o755); err != nil {
			log.Warn().
				Err(err).
				Str("path", path).
				Msg("Failed to create cache directory, skipping read/write mount")
			continue
		}
		bwrapArgs = append(bwrapArgs, "--bind", path, path)
	}

	// For each referenced file/directory and coding-agent config, allow access.
	envVars, filesToMount, err := setupDirMappingsForCodingAgents(config)
	if err != nil {
		return nil, err
	}

	filePathsToMount := config.getDirsToMount()
	filePathsToMount = append(filePathsToMount, filesToMount...)
	for _, dir := range filePathsToMount {
		if dir.readOnly {
			bwrapArgs = append(bwrapArgs, "--ro-bind", dir.hostFilePath, dir.hostFilePath)
		} else {
			bwrapArgs = append(bwrapArgs, "--bind", dir.hostFilePath, dir.hostFilePath)
		}
	}

	for _, envVar := range envVars {
		bwrapArgs = append(bwrapArgs, "--setenv", envVar.key, envVar.value)
	}

	// Run inside the working directory.
	bwrapArgs = append(bwrapArgs, "--chdir", config.workingDir)

	bwrapArgs = append(bwrapArgs, "--")
	bwrapArgs = append(bwrapArgs, cmdArgs...)

	log.Debug().
		Str("args", strings.Join(bwrapArgs, " ")).
		Strs("cmd", bwrapArgs).
		Msg("Running command in native execution mode with bubblewrap")
	return runShellCommand(ctx, bwrapArgs)
}

// ensureBubblewrapCanSandbox runs a tiny bwrap invocation to check that this host
// actually allows bubblewrap to create the namespaces it needs. bwrap must create
// at least a mount namespace to set up its bind mounts, which requires either
// unprivileged user namespaces to be enabled or bwrap to be installed setuid.
// Many environments (Debian with unprivileged_userns_clone=0, Ubuntu 24.04's
// AppArmor restriction, or hardened/nested containers without CAP_SYS_ADMIN)
// disallow this, and bwrap then fails with an opaque "No permissions to create a
// new namespace" error. Probing up front lets us return an actionable message.
func ensureBubblewrapCanSandbox(ctx context.Context) error {
	// This probe uses the same namespace flags as the real invocation, so a
	// success here reliably predicts the real run will get far enough to exec.
	probe := exec.CommandContext(ctx, "bwrap",
		"--ro-bind", "/", "/",
		"--unshare-user-try", "--unshare-pid",
		"--", "true")
	out, err := probe.CombinedOutput()
	if err == nil {
		return nil
	}

	log.Debug().
		Err(err).
		Str("output", strings.TrimSpace(string(out))).
		Msg("bubblewrap sandbox probe failed")

	return fmt.Errorf(
		"bubblewrap cannot create a sandbox on this host (%v): %s\n"+
			"This usually means unprivileged user namespaces are disabled. To fix, either:\n"+
			"  - enable them: `sudo sysctl -w kernel.unprivileged_userns_clone=1` (Debian) or "+
			"`sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0` (Ubuntu 24.04+)\n"+
			"  - install bwrap setuid root, or run on a host/CI runner that permits user namespaces\n"+
			"  - or fall back to Docker with `--mode=docker`",
		err, strings.TrimSpace(string(out)))
}
