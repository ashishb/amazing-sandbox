package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	docker "github.com/fsouza/go-dockerclient"
	isatty "github.com/mattn/go-isatty"
	"github.com/rs/zerolog/log"
)

// RunCmd runs the npx command with the given arguments.
// args can be empty list as well
func RunCmd(ctx context.Context, config Config) error {
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

	// Now run the image with the config
	if err := runDockerContainer1(ctx, config); err != nil {
		return fmt.Errorf("failed to run %s command: %w", config.cmdType, err)
	}
	return nil
}

func checkDockerInstalled(client *docker.Client) error {
	err := client.Ping()
	if err != nil {
		return fmt.Errorf("docker is not running: %w", err)
	}

	log.Debug().Msg("Docker is installed and running")
	return nil
}

func getDockerClient() (*docker.Client, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, fmt.Errorf("docker is not installed: %w", err)
	}
	return client, nil
}

func pullDockerImageIfNotExists(ctx context.Context, client *docker.Client, image string) error {
	_, err := client.InspectImage(image)
	if err == nil {
		log.Debug().
			Str("image", image).
			Msg("Docker image found locally")
		return nil
	}

	if errors.Is(err, docker.ErrNoSuchImage) {
		log.Info().
			Str("image", image).
			Msg("Docker image not found locally, pulling from registry")

		pullOpts := docker.PullImageOptions{
			Context:      ctx,
			Repository:   image,
			OutputStream: os.Stdout,
		}
		authOpts := docker.AuthConfiguration{}

		err = client.PullImage(pullOpts, authOpts)
		if err != nil {
			return fmt.Errorf("failed to pull docker image %s: %w", image, err)
		}

		log.Info().
			Str("image", image).
			Msg("Successfully pulled docker image")
	}

	return nil
}

func runDockerContainer1(ctx context.Context, config Config) error {
	dockerRunCmd, err := getDockerRunCmd(config)
	if err != nil {
		return err
	}

	dockerRunCmd = append(dockerRunCmd, config.args...)
	// fmt.Println(dockerRunCmd)
	log.Debug().
		Strs("dockerRunCmd", dockerRunCmd).
		Msg("Running docker container with command")

	// Execute the docker run command
	// Note: This is a blocking call
	//nolint:gosec  // User is deliberately executing a command
	cmdCtx := exec.CommandContext(ctx, dockerRunCmd[0], dockerRunCmd[1:]...)
	if isInteractiveTerminal() {
		cmdCtx.Stdin = os.Stdin
		cmdCtx.Stdout = os.Stdout
		cmdCtx.Stderr = os.Stderr
	}
	// cmdCtx.Stdout = log.Logger.Level(zerolog.InfoLevel).With().Logger()
	// cmdCtx.Stderr = log.Logger.Level(zerolog.ErrorLevel).With().Strs("dockerRunCmd", dockerRunCmd).Logger()
	err = cmdCtx.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}

	// Check for other errors and return them as-is
	if err != nil {
		return fmt.Errorf("failed to run docker container: %w", err)
	}

	log.Debug().
		Strs("dockerRunCmd", dockerRunCmd).
		Msg("Docker container ran successfully")
	return nil
}

func getDockerRunCmd(config Config) ([]string, error) {
	// If this is an interactive terminal then inform the process about this
	dockerRunCmd := []string{"docker", "run", "--rm", "--init"}
	if isInteractiveTerminal() {
		dockerRunCmd = append(dockerRunCmd, "--interactive", "--tty")
	}

	if config.mountWorkingDirRW {
		dockerRunCmd = append(dockerRunCmd,
			"--mount=type=bind,"+fmt.Sprintf("source=%s,target=%s", config.workingDir, config.workingDir))
	} else if config.mountWorkingDirRO {
		dockerRunCmd = append(dockerRunCmd,
			"--mount=type=bind,"+fmt.Sprintf("source=%s,target=%s,readonly", config.workingDir, config.workingDir))
	}

	if config.getReferencedFiles() != nil {
		for _, dir := range config.getReferencedFiles() {
			if config.mountReferencedDirRW {
				dockerRunCmd = append(dockerRunCmd,
					"--mount=type=bind,"+fmt.Sprintf("source=%s,target=%s", dir, dir))
			} else if config.mountReferencedDirRO {
				dockerRunCmd = append(dockerRunCmd,
					"--mount=type=bind,"+fmt.Sprintf("source=%s,target=%s,readonly", dir, dir))
			}
		}
	}

	if config.loadDotEnv {
		dockerRunCmd = append(dockerRunCmd, "--env-file="+filepath.Join(config.workingDir, ".env"))
	}

	dockerArgs, err := setupDirMappingsForCodingAgents(config)
	if err != nil {
		return nil, err
	}

	dockerRunCmd = append(dockerRunCmd, dockerArgs...)
	dockerRunCmd = append(dockerRunCmd,
		// Warning: without volume names, the volumes are usually deleted when the container is removed
		"--mount=type=volume,src=npm1,target=/.npm",                      // to persist npm cache across runs
		"--mount=type=volume,src=npm2,target=/root/.npm",                 // to persist npm cache across runs
		"--mount=type=volume,src=ruby1,target=/usr/local/bundle/",        // to persist Ruby gem cache across runs
		"--mount=type=volume,src=ruby2,target=/root/.gem/ruby/",          // to persist Ruby gem cache across runs
		"--mount=type=volume,src=ruby3,target=/usr/local/lib/ruby/gems/", // to persist Ruby gem cache across runs
		"--mount=type=volume,src=ruby4,target=/root/.cache/gem/specs",    // to persist Ruby gem cache across runs
		"--mount=type=volume,src=ruby5,target=/root/.rbenv/",             // to persist Ruby gem cache across runs
		"--mount=type=volume,src=cargo1,target=/usr/local/cargo",         // to persist Rust cargo cache across runs
		// to persist pip cache across runs
		"--mount=type=volume,src=pip312,target=/usr/local/lib/python3.12/",
		"--mount=type=volume,src=pip313,target=/usr/local/lib/python3.13/",
		"--mount=type=volume,src=pip314,target=/usr/local/lib/python3.14/",
		"--mount=type=volume,src=pip315,target=/usr/local/lib/python3.15/",
		"--mount=type=volume,src=uv1,target=/root/.cache/uv/",
		"--mount=type=volume,src=poetry1,target=/root/.cache/pypoetry",
		"--net="+string(config.networkType),
		"--workdir="+config.workingDir,
		config.dockerBaseImage)

	// TODO: Use os.Getuid() and os.Getgid() to get the current user and group IDs
	// and run the container as that user if config.runAsNonRoot is true
	return dockerRunCmd, nil
}

func setupDirMappingsForCodingAgents(config Config) ([]string, error) {
	if config.cmdType != CmdTypeNpx {
		return make([]string, 0), nil
	}

	dockerArgs := make([]string, 0)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	claudeConfigFile := filepath.Join(homeDir, ".claude.json")
	if err = touchFile(claudeConfigFile); err != nil {
		return nil, fmt.Errorf("failed to touch %s: %w", claudeConfigFile, err)
	}

	// /tmp/claude.json mapped to /root/.claude.json (inside Docker)
	dockerArgs = append(dockerArgs,
		fmt.Sprintf("--mount=type=bind,src=%s,target=/root/.claude.json", claudeConfigFile))

	dirsToMap := []string{
		".config", // General config directory
		".claude", // Anthropic Claude code config
		".codex",  // OpenAI Codex config
		".gemini", // Google Gemini CLI config
	}

	for _, dirName := range dirsToMap {
		dirPath := filepath.Join(homeDir, dirName)
		if err = os.MkdirAll(dirPath, 0o700); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}

		dockerArgs = append(dockerArgs,
			fmt.Sprintf("--mount=type=bind,src=%s,target=/root/%s", dirPath, dirName))
	}
	return dockerArgs, nil
}

func isInteractiveTerminal() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}

// touchFile mimics the basic behavior of the Unix 'touch' command
func touchFile(name string) error {
	file, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("failed to touch file %s: %w", name, err)
	}

	log.Debug().
		Str("file", name).
		Msg("Created file")
	// It's crucial to close the file to release the file descriptor
	return file.Close()
}
