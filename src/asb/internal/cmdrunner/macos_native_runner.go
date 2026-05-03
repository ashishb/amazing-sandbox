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

		`(allow process-exec (subpath "/bin"))`,
		`(allow process-exec (subpath "/usr/bin"))`,
		`(allow process-exec (subpath "/opt/homebrew"))`,

		// Some more paths to mount
		`(allow file-read* (subpath "/opt/homebrew"))`,
		`(allow file-read* (subpath "/tmp"))`,
		`(allow file-read* (subpath "/var/tmp"))`,
		`(allow file-read* (subpath "/var/folders"))`,
		`(allow file-read* (subpath "/private/var/folders"))`,
		`(allow file-read* (subpath "/private/tmp"))`,
		`(allow file-write* (subpath "/tmp"))`,
		`(allow file-write* (subpath "/var/tmp"))`,
		`(allow file-write* (subpath "/var/folders"))`,
		`(allow file-write* (subpath "/private/var/folders"))`,
		`(allow file-write* (subpath "/private/tmp"))`,
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
		os.ExpandEnv("$HOME/.local"),
		os.ExpandEnv("$HOME/.cache"),
		os.ExpandEnv("$HOME/Library/Caches/Homebrew"),

		// Caches for various package managers
		os.ExpandEnv("$HOME/Library/Caches/pip"),
		os.ExpandEnv("$HOME/.npm"),
		os.ExpandEnv("$HOME/.bun"),
		os.ExpandEnv("$HOME/.gem"),
		os.ExpandEnv("$HOME/.cabal"),
		os.ExpandEnv("$HOME/.cache"),
		os.ExpandEnv("$HOME/.local"),
	}

	roPathsToMount := []string{
		"/opt/homebrew",
		"/Library/Frameworks",
		"/Library/Preferences",
		"/System/Library/Frameworks",
		"/bin",
		"/usr/bin",
		os.ExpandEnv("$HOME/Library/Python"),
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
	log.Info().
		Strs("cmd", cmd).
		Msg("Running command in native execution mode with sandbox-exec")
	return runShellCommand(ctx, cmd)
}
