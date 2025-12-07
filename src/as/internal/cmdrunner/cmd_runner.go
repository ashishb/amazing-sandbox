package cmdrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/rs/zerolog/log"
)

// RunNpxCmd runs the npx command with the given arguments.
// args can be empty list as well
func RunNpxCmd(ctx context.Context, config Config) error {
	// alias npx='docker run --rm --init --user=$(id -u):$(id -g) -v "${PWD}":"${PWD}"
	//--net=host --workdir=${PWD} node:25-bookworm-slim npx'

	client, err := getDockerClient()
	if err != nil {
		return err
	}

	// 1. Check that docker is installed and running
	if err := checkDockerInstalled(client); err != nil {
		return fmt.Errorf("failed to run npx command: %w", err)
	}

	// Download the docker image
	if err := pullDockerImageIfNotExists(ctx, client, config.dockerBaseImage); err != nil {
		return fmt.Errorf("failed to run npx command: %w", err)
	}

	// Now run the image with the config
	if err := runDockerContainer(ctx, client, config); err != nil {
		return fmt.Errorf("failed to run npx command: %w", err)
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
			OutputStream: log.Logger.With().Str("image", image).Logger(),
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

	return fmt.Errorf("failed to inspect docker image %s: %w", image, err)
}

func runDockerContainer(ctx context.Context, client *docker.Client, config Config) error {
	// TODO: add options to load .env file
	var mounts []docker.Mount
	if config.mountWorkingDirRW || config.mountWorkingDirRO {
		mount := docker.Mount{
			Source:      config.workingDir,
			Destination: config.workingDir,
			RW:          false,
		}
		mounts = append(mounts, mount)
	}

	opts := docker.CreateContainerOptions{
		Name:     "",
		Platform: "",
		Config: &docker.Config{
			Image:           config.dockerBaseImage,
			WorkingDir:      config.workingDir,
			NetworkDisabled: config.networkType == NetworkNone,
			// Volumes:    nil,
			Mounts:       mounts,
			AttachStdout: true,
			AttachStderr: true,
		},
		HostConfig:       nil,
		NetworkingConfig: nil,
		Context:          ctx,
	}

	container, err := client.CreateContainer(opts)
	if err != nil {
		return fmt.Errorf("failed to create docker container: %w", err)
	}

	log.Debug().
		Str("containerId", container.ID).
		Msg("Docker container created successfully")

	// Start the container
	err = client.StartContainer(container.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to start docker container: %w", err)
	}

	log.Debug().
		Str("containerId", container.ID).
		Msg("Docker container started successfully")
	if err = client.AttachToContainer(docker.AttachToContainerOptions{
		Container:    container.ID,
		OutputStream: log.Logger.With().Str("containerId", container.ID).Logger(),
		ErrorStream:  log.Logger.With().Str("containerId", container.ID).Logger(),
		Stdout:       true,
		Stderr:       true,
		Stream:       true,
	}); err != nil {
		return fmt.Errorf("failed to attach to docker container: %w", err)
	}

	var outputBuf bytes.Buffer
	var errorBuf bytes.Buffer
	err = client.Logs(docker.LogsOptions{
		Context:           ctx,
		Container:         container.ID,
		OutputStream:      &outputBuf,
		ErrorStream:       &errorBuf,
		InactivityTimeout: 10 * time.Second,
		Tail:              "",
		Since:             0,
		Follow:            false,
		Stdout:            true,
		Stderr:            true,
		Timestamps:        false,
		RawTerminal:       false,
	})
	if err != nil {
		return fmt.Errorf("failed to get logs from docker container: %w", err)
	}

	if outputBuf.Len() > 0 {
		log.Info().
			Str("containerId", container.ID).
			Msgf("Docker container logs:\n%s", outputBuf.String())
	}

	if errorBuf.Len() > 0 {
		log.Error().
			Str("containerId", container.ID).
			Msgf("Docker container error logs:\n%s", errorBuf.String())
	}

	return nil
}
