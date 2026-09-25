//go:build darwin

package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

func runCmdInNative(ctx context.Context, config Config) (*ShellResult, error) {
	// config.args is already normalized by SetArgs (e.g. ["node", "index.js"]).
	cmdArgs := config.args
	log.Debug().
		Str("args", strings.Join(cmdArgs, " ")).
		Strs("cmdArgs", cmdArgs).
		Msg("Running command in native execution mode")

	// Ref: https://igorstechnoclub.com/sandbox-exec/
	sandboxConfig := []string{
		"(version 1)",
		"(deny default)",
		"(allow sysctl-read)",
		`(allow file-read* (literal "/"))`,
		"(allow file-read-metadata)",
		"(deny file-write*)",
		"(deny file-read*)",
		// Explicitly allow process forking as "gem" needs it
		`(allow process-fork)`,

		`(allow process-exec (subpath "/bin"))`,
		`(allow process-exec (subpath "/usr/bin"))`,
		`(allow process-exec (subpath "/opt/homebrew"))`,
		`(allow process-exec (subpath "/Library/Frameworks/Python.framework"))`,
		// GitHub Actions put tools inside $HOME/hostedtoolcache
		// e.g. uv is in /Users/runner/hostedtoolcache/uv/0.11.16/aarch64/uv
		// Ref: https://devopsdirective.com/posts/2025/07/github-actions-tool-cache/
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/hostedtoolcache")),
		// Some package managers on GitHub Actions install binaries in the home directory, so allow executing from there as well
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/work/_temp")),
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/.bun")),
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/.local/venv")),
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/.npm/")),
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/.rbenv")),
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/.rustup")),
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/.yarn")),
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/setup-pnpm")),

		// Some more paths to mount
		`(allow file-read* (subpath "/opt/homebrew"))`,
		`(allow file-read* (subpath "/tmp"))`,
		`(allow file-read* (subpath "/var/tmp"))`,
		`(allow file-read* (subpath "/var/folders"))`,
		`(allow file-read* (subpath "/private/var/folders"))`,
		`(allow file-read* (subpath "/private/tmp"))`,
		`(allow file-read* (literal "/dev/null"))`,
		`(allow file-read* (literal "/dev/random"))`,
		`(allow file-read* (literal "/dev/tty"))`,
		`(allow file-read* (literal "/private/etc/ssl/openssl.cnf"))`, // Required for cargo
		`(allow file-write* (subpath "/tmp"))`,
		`(allow file-write* (subpath "/var/tmp"))`,
		`(allow file-write* (subpath "/var/folders"))`,
		`(allow file-write* (subpath "/private/var/folders"))`,
		`(allow file-write* (subpath "/private/tmp"))`,
		`(allow file-write* (literal "/dev/null"))`,
		// To get user info allow read access to /etc/passwd.
		`(allow file-read-data (subpath "/usr/lib/"))`,
		`(allow file-read-data (literal "/private/etc/passwd"))`,
		// For timezone information allow reading these files
		`(allow file-read-data (subpath "/usr/share/locale/"))`,
		`(allow file-read-data (subpath "/private/var/db/timezone"))`,
		`(allow file-read-data (subpath "/usr/share/icu/"))`,

		// For dtrace support, allow access to dtracehelper
		`(allow file-ioctl (literal "/dev/dtracehelper"))`,

		// Needed for user information
		`(allow mach-lookup (global-name "com.apple.SystemConfiguration.configd"))`,
		`(allow mach-lookup (global-name "com.apple.system.opendirectoryd.libinfo"))`,
		`(allow mach-lookup (global-name "com.apple.logd"))`,
		`(allow mach-lookup (global-name "com.apple.system.notification_center"))`,
		`(allow mach-lookup (global-name "com.apple.diagnosticd"))`,
		`(allow mach-lookup (global-name "com.apple.pasteboard.1"))`,
		`(allow mach-lookup (global-name "com.apple.tccd.system"))`,
		`(allow mach-lookup (global-name "com.apple.windowserver.active"))`,
		`(allow mach-lookup (global-name "com.apple.DiskArbitration.diskarbitrationd"))`,
		`(allow mach-lookup (global-name "com.apple.CoreServices.coreservicesd"))`,
		`(allow ipc-posix-shm-read-data (ipc-posix-name "apple.shm.notification_center"))`,
	}
	if config.networkType == NetworkNone {
		sandboxConfig = append(sandboxConfig, "(deny network*)", "(deny system-socket)")
	} else {
		sandboxConfig = append(sandboxConfig, "(allow network*)", "(allow system-socket)")
	}

	rwPathsToMount := []string{
		"/tmp",
		"/var/tmp",
		"/var/folders",
		"/private/var/folders",

		"/dev/dtracehelper", // Ref: https://apple.stackexchange.com/questions/384593/apple-dtracehelper-file
		os.ExpandEnv("$HOME/Library/Caches/Homebrew"),
		os.ExpandEnv("$HOME/Library/Developer/Xcode"),
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

	roPathsToMount := []string{
		"/bin",
		"/opt/homebrew",
		"/usr/bin",
		"/Applications/", // For executing apps like Xcode, Safari, etc.
		"/Library/Developer/CommandLineTools",
		"/Library/Frameworks",
		"/Library/Preferences",
		"/System/Library/Frameworks",
		"/System/Library/Preferences/Logging",
		"/System/Volumes/Preboot/Cryptexes/OS",
		"/System/Library/CoreServices/",
		"/Library/Application Support/CrashReporter/DiagnosticMessagesHistory.plist", // For Zig to decide on crash reporting
		os.ExpandEnv("$HOME/Library/Python"),
		os.ExpandEnv("$HOME/Library/Preferences"),
		// GitHub Actions put tools inside $HOME/hostedtoolcache
		// e.g. uv is in /Users/runner/hostedtoolcache/uv/0.11.16/aarch64/uv
		// Ref: https://devopsdirective.com/posts/2025/07/github-actions-tool-cache/
		os.ExpandEnv("$HOME/hostedtoolcache"),
		// Some package managers on GitHub Actions install binaries in the home directory, so allow executing from there as well
		os.ExpandEnv("$HOME/work/_temp"),
		os.ExpandEnv("$HOME/setup-pnpm"), // For pnpm
	}

	// For each referenced file/directory, we need to allow read access to it
	_, filesToMount, err := setupDirMappingsForCodingAgents(config)
	if err != nil {
		return nil, err
	}

	filePathsToMount := config.getDirsToMount()
	filePathsToMount = append(filePathsToMount, filesToMount...)
	for _, dir := range filePathsToMount {
		mountStr := make([]string, 0, 1)
		if dir.readOnly {
			mountStr = append(mountStr, fmt.Sprintf(`(allow file-read* (subpath "%s"))`, dir.hostFilePath))
		} else {
			mountStr = append(mountStr, fmt.Sprintf(`(allow file-read* (subpath "%s"))`, dir.hostFilePath))
			mountStr = append(mountStr, fmt.Sprintf(`(allow file-write* (subpath "%s"))`, dir.hostFilePath))
		}
		sandboxConfig = append(sandboxConfig, mountStr...)
	}

	for _, path := range roPathsToMount {
		sandboxConfig = append(sandboxConfig, fmt.Sprintf(`(allow file-read* (subpath "%s"))`, path))
	}
	for _, path := range rwPathsToMount {
		sandboxConfig = append(sandboxConfig, fmt.Sprintf(`(allow file-read* (subpath "%s"))`, path))
		sandboxConfig = append(sandboxConfig, fmt.Sprintf(`(allow file-write* (subpath "%s"))`, path))
	}

	if config.cmdType == CmdTypeExec {
		// The binary may live outside the paths allowed above (e.g. ~/bin), so
		// allow reading and executing it. sandbox-exec checks the real path, so
		// allow both the path as invoked and the symlink-resolved one.
		absPath, realPath, err := resolveExecBinary(config)
		if err != nil {
			return nil, err
		}
		for _, path := range []string{absPath, realPath} {
			sandboxConfig = append(sandboxConfig,
				fmt.Sprintf(`(allow file-read* (literal "%s"))`, path),
				fmt.Sprintf(`(allow process-exec (literal "%s"))`, path))
		}
		cmdArgs = append([]string{absPath}, cmdArgs[1:]...)
	}

	if len(cmdArgs) == 0 {
		return nil, errors.New("no command to run in native mode")
	}

	cmd := make([]string, 0, 3+len(cmdArgs))
	cmd = append(cmd, "sandbox-exec", "-p", strings.Join(sandboxConfig, "\n"))
	// One can see the config with
	// os.WriteFile("/tmp/debug.sb", []byte(strings.Join(sandboxConfig, "\n")), 0o644)
	cmd = append(cmd, cmdArgs...)
	log.Debug().
		Strs("cmd", cmd).
		Msg("Running command in native execution mode with sandbox-exec")
	return runShellCommand(ctx, cmd)
}
