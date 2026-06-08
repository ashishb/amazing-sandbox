package cmdrunner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

func runCmdInDocker(ctx context.Context, config Config) error {
	client, err := getDockerClient()
	if err != nil {
		return err
	}

	// 1. Check that docker is installed and running
	if err := checkDockerInstalled(client); err != nil {
		return fmt.Errorf("failed to run %s command: %w", config.cmdType, err)
	}

	// Download the docker image
	if err := pullDockerImageIfNotExists(ctx, client, config.dockerBaseImage); err != nil {
		return fmt.Errorf("failed to run %s command: %w", config.cmdType, err)
	}

	if err := runDockerContainer(ctx, config); err != nil {
		return fmt.Errorf("failed to run %s command: %w", config.cmdType, err)
	}
	return nil
}

func runDockerContainer(ctx context.Context, config Config) error {
	dockerRunCmd, err := getDockerRunCmd(config)
	if err != nil {
		return err
	}

	dockerRunCmd = append(dockerRunCmd, config.args...)
	// fmt.Println(dockerRunCmd)
	log.Debug().
		Strs("dockerRunCmd", dockerRunCmd).
		Msg("Running docker container with command")

	return runShellCommand(ctx, dockerRunCmd)
}

func getDockerRunCmd(config Config) ([]string, error) {
	// If this is an interactive terminal then inform the process about this
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

	dockerRunCmd = append(dockerRunCmd,
		// Warning: without volume names, the volumes are usually deleted when the container is removed
		"--mount=type=volume,src=npm1,target=/.npm",                       // to persist npm cache across runs
		"--mount=type=volume,src=npm2,target=/root/.npm",                  // to persist npm cache across runs
		"--mount=type=volume,src=npm3,target=/usr/local/lib/node_modules", // to persist "npm install -g" cache across runs
		"--mount=type=volume,src=bun1,target=/root/.bun/install/cache",    // to persist bun cache across runs
		"--mount=type=volume,src=ruby1,target=/usr/local/bundle/",         // to persist Ruby gem cache across runs
		"--mount=type=volume,src=ruby2,target=/root/.gem/ruby/",           // to persist Ruby gem cache across runs
		"--mount=type=volume,src=ruby3,target=/usr/local/lib/ruby/gems/",  // to persist Ruby gem cache across runs
		"--mount=type=volume,src=ruby4,target=/root/.cache/gem/specs",     // to persist Ruby gem cache across runs
		"--mount=type=volume,src=ruby5,target=/root/.rbenv/",              // to persist Ruby gem cache across runs
		"--mount=type=volume,src=cargo1,target=/usr/local/cargo",          // to persist Rust cargo cache across runs
		"--mount=type=volume,src=cabal1,target=/root/.cabal/",             // to persist Haskell cabal cache across runs
		"--mount=type=volume,src=go1,target=/go",                          // to persist Go module cache across runs
		"--mount=type=volume,src=go2,target=/root/.cache/go-build",        // to persist Go build cache across runs

		// to persist pip cache across runs
		"--mount=type=volume,src=pip312,target=/usr/local/lib/python3.12/",
		"--mount=type=volume,src=pip313,target=/usr/local/lib/python3.13/",
		"--mount=type=volume,src=pip314,target=/usr/local/lib/python3.14/",
		"--mount=type=volume,src=pip315,target=/usr/local/lib/python3.15/",
		"--mount=type=volume,src=uv1,target=/root/.cache/uv/",
		"--mount=type=volume,src=uv2,target=/root/.local/share/uv/",
		"--mount=type=volume,src=poetry1,target=/root/.cache/pypoetry",
		"--mount=type=volume,src=zig1,target=/root/.cache/zig", // to persist Zig cache across runs
		"--mount=type=volume,src=zig2,target=/root/.zig-cache", // to persist Zig cache across runs
		"--network="+string(config.networkType),
		"--workdir="+config.workingDir,
		config.dockerBaseImage)

	// TODO: Use os.Getuid() and os.Getgid() to get the current user and group IDs
	// and run the container as that user if config.runAsNonRoot is true
	return dockerRunCmd, nil
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
