package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog/log"

	docker "github.com/fsouza/go-dockerclient"
)

// This can be a file or a directory to mount into the Docker container.
type _FilePathToMount struct {
	hostFilePath      string
	containerFilePath string // Usually, same as hostFilePath
	readOnly          bool
}

func newFilePathToMount(hostDir, containerDir string, readOnly bool) _FilePathToMount {
	return _FilePathToMount{
		hostFilePath:      hostDir,
		containerFilePath: containerDir,
		readOnly:          readOnly,
	}
}

type EnvVar struct {
	key   string
	value string
}

// RunCmd runs the npx command with the given arguments.
// args can be empty list as well
func RunCmd(ctx context.Context, config Config) error {
	// Now run the image with the config
	switch config.execMode {
	case ExecModeDocker:
		return runCmdInDocker(ctx, config)
	case ExecModeNative:
		log.Fatal().Msg("Running in native execution mode is not supported yet")
	}
	return nil
}

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
