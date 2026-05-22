package cmdrunner

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

func runCmdInNative(ctx context.Context, config Config) error {
	cmdArgs := config.cmdType.getArgs(config.args)
	log.Debug().
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
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/.rbenv")),
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/.rustup")),
		fmt.Sprintf(`(allow process-exec (subpath "%s"))`, os.ExpandEnv("$HOME/.yarn")),

		// Some more paths to mount
		`(allow file-read* (subpath "/opt/homebrew"))`,
		`(allow file-read* (subpath "/tmp"))`,
		`(allow file-read* (subpath "/var/tmp"))`,
		`(allow file-read* (subpath "/var/folders"))`,
		`(allow file-read* (subpath "/private/var/folders"))`,
		`(allow file-read* (subpath "/private/tmp"))`,
		`(allow file-read* (literal "/dev/null"))`,
		`(allow file-read* (literal "/dev/random"))`,
		`(allow file-read* (literal "/private/etc/ssl/openssl.cnf"))`, // Required for cargo
		`(allow file-write* (subpath "/tmp"))`,
		`(allow file-write* (subpath "/var/tmp"))`,
		`(allow file-write* (subpath "/var/folders"))`,
		`(allow file-write* (subpath "/private/var/folders"))`,
		`(allow file-write* (subpath "/private/tmp"))`,
		`(allow file-write* (literal "/dev/null"))`,
		// To get user info allow read access to /etc/passwd.
		`(allow file-read-data (literal "/private/etc/passwd"))`,
		// For timezone information allow reading these files
		`(allow file-read-data (subpath "/usr/share/locale/"))`,
		`(allow file-read-data (subpath "/private/var/db/timezone"))`,

		// Needed for user information
		`(allow mach-lookup (global-name "com.apple.SystemConfiguration.configd"))`,
		`(allow mach-lookup (global-name "com.apple.system.opendirectoryd.libinfo"))`,
		`(allow mach-lookup (global-name "com.apple.logd"))`,
		`(allow mach-lookup (global-name "com.apple.system.notification_center"))`,
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
		os.ExpandEnv("$HOME/Library/Caches/Homebrew"),
		os.ExpandEnv("$HOME/Library/Developer/Xcode"),

		// Caches for various package managers
		os.ExpandEnv("$HOME/Library/Caches/pip"),
		os.ExpandEnv("$HOME/.bun"),
		os.ExpandEnv("$HOME/.cabal"),
		os.ExpandEnv("$HOME/.cache"),
		os.ExpandEnv("$HOME/.cargo"),
		os.ExpandEnv("$HOME/.gem"),
		os.ExpandEnv("$HOME/.local"),
		os.ExpandEnv("$HOME/.npm"),
		os.ExpandEnv("$HOME/.rustup"),
		os.ExpandEnv("$HOME/.rbenv"),
		os.ExpandEnv("$HOME/.yarn"),
	}

	roPathsToMount := []string{
		"/bin",
		"/dev/dtracehelper",
		"/opt/homebrew",
		"/usr/bin",
		"/Library/Developer/CommandLineTools",
		"/Library/Frameworks",
		"/Library/Preferences",
		"/System/Library/Frameworks",
		"/System/Library/Preferences/Logging",
		"/System/Volumes/Preboot/Cryptexes/OS",
		os.ExpandEnv("$HOME/Library/Python"),
		os.ExpandEnv("$HOME/Library/Preferences"),
	}

	// For each referenced file/directory, we need to allow read access to it
	_, filesToMount, err := setupDirMappingsForCodingAgents(config)
	if err != nil {
		return err
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

	cmd := make([]string, 0, 3+len(cmdArgs)-1)
	cmd = append(cmd, "sandbox-exec", "-p", strings.Join(sandboxConfig, "\n"))
	// One can see the config with
	// os.WriteFile("/tmp/debug.sb", []byte(strings.Join(sandboxConfig, "\n")), 0o644)
	cmd = append(cmd, cmdArgs[1:]...)
	log.Debug().
		Strs("cmd", cmd).
		Msg("Running command in native execution mode with sandbox-exec")
	return runShellCommand(ctx, cmd)
}
