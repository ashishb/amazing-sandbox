package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"os"

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
		return runCmdInNative(ctx, config)
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
