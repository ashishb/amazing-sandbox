package cmdrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	isatty "github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
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

	return nil
}

func runDockerContainer1(ctx context.Context, config Config) error {
	// If this is an interactive terminal then inform the process about this
	isInteractiveTerminal := isatty.IsTerminal(os.Stdin.Fd())
	dockerRunCmd := []string{"docker", "run", "--rm", "--init"}
	if isInteractiveTerminal {
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
			} else {
				dockerRunCmd = append(dockerRunCmd,
					"--mount=type=bind,"+fmt.Sprintf("source=%s,target=%s,readonly", dir, dir))
			}
		}
	}

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
		"--net="+string(config.networkType),
		"--workdir="+config.workingDir,
		config.dockerBaseImage)

	dockerRunCmd = append(dockerRunCmd, config.args...)
	// fmt.Println(dockerRunCmd)
	log.Debug().
		Strs("dockerRunCmd", dockerRunCmd).
		Msg("Running docker container with command")

	// Execute the docker run command
	// Note: This is a blocking call
	cmdCtx := exec.CommandContext(ctx, dockerRunCmd[0], dockerRunCmd[1:]...)
	if isInteractiveTerminal {
		cmdCtx.Stdin = os.Stdin
	}
	cmdCtx.Stdout = log.Logger.Level(zerolog.InfoLevel).With().Logger()
	cmdCtx.Stderr = log.Logger.Level(zerolog.ErrorLevel).With().Strs("dockerRunCmd", dockerRunCmd).Logger()
	cmd := cmdCtx.Run()
	if cmd != nil {
		return fmt.Errorf("failed to run docker container: %w", cmd)
	}

	log.Debug().
		Strs("dockerRunCmd", dockerRunCmd).
		Msg("Docker container ran successfully")
	return nil
}

func getCurrentUserID() int {
	return os.Getuid()
}

func getCurrentGroupID() int {
	return os.Getgid()
}

// This is the proper function to run the docker container except I am unable to see the logs right now
// via this and that has to be debugged.
func runDockerContainer2(ctx context.Context, client *docker.Client, config Config) error {
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
		OutputStream: log.Logger.With().Str("containerId", container.ID).Logger().Level(zerolog.InfoLevel),
		ErrorStream:  log.Logger.With().Str("containerId", container.ID).Logger().Level(zerolog.ErrorLevel),
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
