package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// Select based on executable availability, not daemon health. A failing runtime
// invocation must not silently switch runtimes (and their separate caches).
func selectContainerRuntime(lookPath func(string) (string, error)) (ExecMode, error) {
	for _, mode := range []ExecMode{ExecModeDocker, ExecModePodman} {
		if _, err := lookPath(string(mode)); err == nil {
			return mode, nil
		}
	}
	return "", errors.New("neither podman nor docker is available in PATH; install a container runtime or use --mode=native")
}

func runPodmanContainer(ctx context.Context, config Config) (*ShellResult, error) {
	// Podman's CLI works without a Docker-compatible API socket. Fully qualify
	// Docker Hub images to avoid Podman's interactive short-name resolution.
	if config.dockerBaseImage == _dockerImageMap[config.cmdType] {
		config.dockerBaseImage = qualifyPodmanImage(config.dockerBaseImage)
	}
	cmd, err := getContainerRunCmd(config, ExecModePodman)
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, config.args...)
	log.Debug().Strs("podmanRunCmd", cmd).Msg("Running podman container with command")
	// `podman run` pulls missing images automatically and reuses local images.
	result, err := runShellCommand(ctx, cmd)
	if err != nil {
		return result, fmt.Errorf("failed to run %s command with podman: %w", config.cmdType, err)
	}
	return result, nil
}

func qualifyPodmanImage(image string) string {
	first, _, hasSlash := strings.Cut(image, "/")
	if hasSlash && (strings.ContainsAny(first, ".:") || first == "localhost") {
		return image
	}
	if !hasSlash {
		return "docker.io/library/" + image
	}
	return "docker.io/" + image
}
