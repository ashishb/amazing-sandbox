package cmdrunner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func runCmdInDocker(ctx context.Context, config Config) (*ShellResult, error) {
	client, err := getDockerClient()
	if err != nil {
		return nil, err
	}

	// 1. Check that docker is installed and running
	if err := checkDockerInstalled(client); err != nil {
		return nil, fmt.Errorf("failed to run %s command: %w", config.cmdType, err)
	}

	// Download the docker image
	if err := pullDockerImageIfNotExists(ctx, client, config.dockerBaseImage); err != nil {
		return nil, fmt.Errorf("failed to run %s command: %w", config.cmdType, err)
	}

	if result, err := runDockerContainer(ctx, config); err != nil {
		return result, fmt.Errorf("failed to run %s command: %w", config.cmdType, err)
	}
	return nil, nil
}

// networkSnapshot holds total rx/tx byte counts across all non-loopback
// network interfaces at a single point in time, read from /proc/net/dev.
type networkSnapshot struct {
	rxBytes uint64
	txBytes uint64
}

// readNetworkSnapshot parses /proc/net/dev and returns the aggregate rx/tx
// byte totals across all non-loopback interfaces. It returns an error when
// /proc/net/dev is not available (e.g., macOS / Windows).
func readNetworkSnapshot() (networkSnapshot, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return networkSnapshot{}, fmt.Errorf("cannot read /proc/net/dev: %w", err)
	}
	var snap networkSnapshot
	for _, line := range strings.Split(string(data), "\n") {
		colonIdx := strings.IndexByte(line, ':')
		if colonIdx < 0 {
			continue
		}
		iface := strings.TrimSpace(line[:colonIdx])
		if iface == "lo" {
			continue // skip loopback
		}
		// /proc/net/dev columns after the colon:
		//   [0]  rx_bytes  [1-7] other rx fields  [8]  tx_bytes  [9-15] other tx fields
		fields := strings.Fields(line[colonIdx+1:])
		if len(fields) < 9 {
			continue
		}
		rxB, err1 := strconv.ParseUint(fields[0], 10, 64)
		txB, err2 := strconv.ParseUint(fields[8], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		snap.rxBytes += rxB
		snap.txBytes += txB
	}
	return snap, nil
}

func runDockerContainer(ctx context.Context, config Config) (*ShellResult, error) {
	dockerRunCmd, err := getDockerRunCmd(config)
	if err != nil {
		return nil, err
	}

	dockerRunCmd = append(dockerRunCmd, config.args...)
	log.Debug().
		Strs("dockerRunCmd", dockerRunCmd).
		Msg("Running docker container with command")

	// Snapshot network counters before and after the container run.
	// /proc/net/dev works for --network=host (default) and --network=bridge.
	// It is unavailable on macOS/Windows; in that case we skip the output.
	beforeSnap, snapErr := readNetworkSnapshot()
	result, runErr := runShellCommand(ctx, dockerRunCmd)
	afterSnap, _ := readNetworkSnapshot()

	if snapErr != nil {
		log.Debug().Err(snapErr).Msg("Network usage stats unavailable on this platform")
	} else {
		var rxDelta, txDelta uint64
		if afterSnap.rxBytes >= beforeSnap.rxBytes {
			rxDelta = afterSnap.rxBytes - beforeSnap.rxBytes
		}
		if afterSnap.txBytes >= beforeSnap.txBytes {
			txDelta = afterSnap.txBytes - beforeSnap.txBytes
		}
		fmt.Fprintf(os.Stderr, "\nNetwork usage: received %s, sent %s\n",
			formatBytes(rxDelta), formatBytes(txDelta))
	}

	return result, runErr
}

// formatBytes returns a human-readable byte count using SI decimal prefixes
// (1 kB = 1000 B), matching Docker's own network I/O display convention.
func formatBytes(n uint64) string {
	const unit = uint64(1000)
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div := unit
	exp := 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "kMGTPE"[exp])
}

func getDockerRunCmd(config Config) ([]string, error) {
	// If this is an interactive terminal then inform the process about this.
	dockerRunCmd := []string{"docker", "run", "--rm", "--init"}
	if isInteractiveTerminal() {
		dockerRunCmd = append(dockerRunCmd, "--interactive", "--tty")
	}

	envVars, filesToMount, err := setupDirMappingsForCodingAgents(config)
	if err != nil {
		return nil, err
	}

	for _, envVar := range envVars {
		dockerRunCmd = append(dockerRunCmd, "--env="+envVar.key+"="+envVar.value)
	}

	filePathsToMount := config.getDirsToMount()
	filePathsToMount = append(filePathsToMount, filesToMount...)
	for _, dir := range filePathsToMount {
		var mountStr string
		if dir.readOnly {
			mountStr = fmt.Sprintf("source=%s,target=%s,readonly", dir.hostFilePath, dir.containerFilePath)
		} else {
			mountStr = fmt.Sprintf("source=%s,target=%s", dir.hostFilePath, dir.containerFilePath)
		}
		dockerRunCmd = append(dockerRunCmd, "--mount=type=bind,"+mountStr)
	}

	if config.loadDotEnv {
		dockerRunCmd = append(dockerRunCmd, "--env-file="+filepath.Join(config.workingDir, ".env"))
	}

	for volumeName, containerPath := range getCacheMounts() {
		// Warning: without volume names, the volumes are usually deleted when the container is removed
		dockerRunCmd = append(dockerRunCmd,
			fmt.Sprintf("--mount=type=volume,src=%s,target=%s", volumeName, containerPath))
	}

	dockerRunCmd = append(dockerRunCmd,
		"--network="+string(config.networkType),
		"--workdir="+config.workingDir,
		config.dockerBaseImage)

	// TODO: Use os.Getuid() and os.Getgid() to get the current user and group IDs
	// and run the container as that user if config.runAsNonRoot is true
	return dockerRunCmd, nil
}

type (
	_VolumeName        = string
	_ContainerFilePath = string
)

func getCacheMounts() map[_VolumeName]_ContainerFilePath {
	return map[_VolumeName]_ContainerFilePath{
		// Javascript/TypeScript
		"npm1": "/.npm",
		"npm2": "/root/.npm",
		"npm3": "/usr/local/lib/node_modules", // to persist "npm install -g" cache across runs
		"bun1": "/root/.bun/install/cache",

		// Ruby
		"ruby1": "/usr/local/bundle/",
		"ruby2": "/root/.gem/ruby/",
		"ruby3": "/usr/local/lib/ruby/gems/",
		"ruby4": "/root/.cache/gem/specs",
		"ruby5": "/root/.rbenv/",

		"cargo1": "/usr/local/cargo",      // Rust
		"cabal1": "/root/.cabal/",         // Haskell
		"go1":    "/go",                   // Go module cache
		"go2":    "/root/.cache/go-build", // Go build cache

		// Python cache
		"pip312":  "/usr/local/lib/python3.12/",
		"pip313":  "/usr/local/lib/python3.13/",
		"pip314":  "/usr/local/lib/python3.14/",
		"pip315":  "/usr/local/lib/python3.15/",
		"uv1":     "/root/.cache/uv/",
		"uv2":     "/root/.local/share/uv/",
		"poetry1": "/root/.cache/pypoetry",

		"zig1": "/root/.cache/zig", // to persist Zig cache across runs
		"zig2": "/root/.zig-cache", // to persist Zig build cache across runs
	}
}

func setupDirMappingsForCodingAgents(config Config) ([]EnvVar, []_FilePathToMount, error) {
	if !slices.Contains([]CmdType{CmdTypeBash, CmdTypeNpx}, config.cmdType) {
		log.Debug().
			Str("cmdType", string(config.cmdType)).
			Msg("Not likely a coding agent command, skipping coding agent directory mounting")
		return nil, nil, nil
	}

	if !config.mountReferencedDirRW && !config.mountReferencedDirRO {
		log.Debug().
			Msg("No disk access enabled inside the sandbox, skipping directory mappings for coding agents")
		return nil, nil, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	claudeConfigFile := filepath.Join(homeDir, ".claude.json")
	if err = touchFile(claudeConfigFile); err != nil {
		return nil, nil, fmt.Errorf("failed to touch %s: %w", claudeConfigFile, err)
	}

	filesToMounts := make([]_FilePathToMount, 0)
	// For claude add IS_SANDBOX=1 (https://github.com/ashishb/amazing-sandbox/issues/16)
	envVars := []EnvVar{
		{key: "IS_SANDBOX", value: "1"},
	}

	// $HOME/claude.json mapped to /root/.claude.json (inside Docker)
	filesToMounts = append(filesToMounts, newFilePathToMount(claudeConfigFile, "/root/.claude.json", false))

	dirsToMap := []string{
		".config", // General config directory
		".claude", // Anthropic Claude code config
		".codex",  // OpenAI Codex config
		".gemini", // Google Gemini CLI config

		// xAI Grok binary is stored at `~/.grok/bin`
		".grok", // xAI Grok config - https://docs.x.ai/build/settings
	}

	for _, dirName := range dirsToMap {
		dirPath := filepath.Join(homeDir, dirName)
		if err = os.MkdirAll(dirPath, 0o700); err != nil {
			return nil, nil, fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}

		if config.mountReferencedDirRO {
			filesToMounts = append(filesToMounts, newFilePathToMount(dirPath, "/root/"+dirName, true))
		} else {
			filesToMounts = append(filesToMounts, newFilePathToMount(dirPath, "/root/"+dirName, false))
		}
	}
	return envVars, filesToMounts, nil
}
