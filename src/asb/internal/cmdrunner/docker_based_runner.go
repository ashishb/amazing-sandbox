package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	docker "github.com/fsouza/go-dockerclient"
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

	if result, err := runDockerContainer(ctx, client, config); err != nil {
		return result, fmt.Errorf("failed to run %s command: %w", config.cmdType, err)
	}
	return nil, nil
}

func runDockerContainer(ctx context.Context, client *docker.Client, config Config) (*ShellResult, error) {
	containerName := fmt.Sprintf("asb-%d", time.Now().UnixNano())

	dockerRunCmd, err := getDockerRunCmd(config, containerName)
	if err != nil {
		return nil, err
	}

	dockerRunCmd = append(dockerRunCmd, config.args...)
	log.Debug().
		Strs("dockerRunCmd", dockerRunCmd).
		Msg("Running docker container with command")

	statsCtx, statsCancel := context.WithCancel(ctx)
	var rxBytes, txBytes uint64
	statsGoroutineDone := make(chan struct{})

	go func() {
		defer close(statsGoroutineDone)
		collectContainerNetworkStats(statsCtx, client, containerName, &rxBytes, &txBytes)
	}()

	result, runErr := runShellCommand(ctx, dockerRunCmd)

	statsCancel()
	<-statsGoroutineDone

	fmt.Fprintf(os.Stderr, "\nNetwork usage: received %s, sent %s\n",
		formatBytes(rxBytes), formatBytes(txBytes))

	if removeErr := client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    containerName,
		Force: true,
	}); removeErr != nil {
		log.Debug().Err(removeErr).Str("container", containerName).Msg("Failed to remove container")
	}

	return result, runErr
}

// collectContainerNetworkStats streams Docker stats for the named container and
// writes the cumulative rx/tx byte totals into rxBytes and txBytes. It retries
// if the container has not yet started, and stops when ctx is cancelled.
func collectContainerNetworkStats(ctx context.Context, client *docker.Client, containerName string, rxBytes, txBytes *uint64) {
	for {
		statsCh := make(chan *docker.Stats, 10)
		errCh := make(chan error, 1)

		go func(ch chan<- *docker.Stats) {
			errCh <- client.Stats(docker.StatsOptions{
				ID:      containerName,
				Stats:   ch,
				Stream:  true,
				Context: ctx,
			})
		}(statsCh)

		var lastStats *docker.Stats
		for s := range statsCh {
			if s != nil {
				lastStats = s
			}
		}

		statsErr := <-errCh

		// Context was cancelled (container run finished): use last collected stats.
		if ctx.Err() != nil {
			if lastStats != nil {
				sumNetworkStats(lastStats, rxBytes, txBytes)
			}
			return
		}

		if statsErr == nil {
			// Streaming ended normally (container removed while still running).
			if lastStats != nil {
				sumNetworkStats(lastStats, rxBytes, txBytes)
			}
			return
		}

		// Container not found yet – retry after a short delay.
		var nsErr *docker.NoSuchContainer
		if !errors.As(statsErr, &nsErr) {
			log.Debug().Err(statsErr).Msg("Unexpected error collecting container stats")
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// sumNetworkStats aggregates rx/tx bytes across all networks in a Stats snapshot.
func sumNetworkStats(s *docker.Stats, rxBytes, txBytes *uint64) {
	for _, n := range s.Networks {
		*rxBytes += n.RxBytes
		*txBytes += n.TxBytes
	}
	// Fallback for single-network containers that only populate Network (not Networks).
	if len(s.Networks) == 0 {
		*rxBytes += s.Network.RxBytes
		*txBytes += s.Network.TxBytes
	}
}

// formatBytes returns a human-readable byte count (e.g. "1.2 MB").
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

func getDockerRunCmd(config Config, containerName string) ([]string, error) {
	// If this is an interactive terminal then inform the process about this.
	// Note: --rm is intentionally omitted; the container is removed manually after
	// collecting network stats.
	dockerRunCmd := []string{"docker", "run", "--init", "--name=" + containerName}
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
	if config.cmdType != CmdTypeNpx {
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
