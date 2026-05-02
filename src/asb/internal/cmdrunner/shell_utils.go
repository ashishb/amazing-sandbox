package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

func runShellCommand(ctx context.Context, shellCmd []string) error {
	// Execute the shell command
	// Note: This is a blocking call
	//nolint:gosec  // User is deliberately executing a command
	cmdCtx := exec.CommandContext(ctx, shellCmd[0], shellCmd[1:]...)
	if isInteractiveTerminal() {
		log.Debug().Msg("Running shell command in interactive mode")
		cmdCtx.Stdin = os.Stdin
		cmdCtx.Stdout = os.Stdout
		cmdCtx.Stderr = os.Stderr
	}
	// cmdCtx.Stdout = log.Logger.Level(zerolog.InfoLevel).With().Logger()
	// cmdCtx.Stderr = log.Logger.Level(zerolog.ErrorLevel).With().Strs("shellCmd", shellCmd).Logger()
	err := cmdCtx.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}

	// Check for other errors and return them as-is
	if err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	log.Debug().
		Strs("shellCmd", shellCmd).
		Msg("Command ran successfully")
	return nil
}
