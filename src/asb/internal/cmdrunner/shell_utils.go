package cmdrunner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

type (
	NumBytesStdout = int
	NumBytesStderr = int
)

type ShellResult struct {
	stdoutBytes NumBytesStdout
	stderrBytes NumBytesStderr
}

func (r ShellResult) NumBytesStdOut() int {
	return r.stdoutBytes
}

func (r ShellResult) NumBytesStdErr() int {
	return r.stderrBytes
}

type _CountingWriter struct {
	writer io.Writer
	count  int
}

func newCountingWriter(w io.Writer) *_CountingWriter {
	return &_CountingWriter{writer: w, count: 0}
}

func (cw *_CountingWriter) Write(p []byte) (int, error) {
	n, err := cw.writer.Write(p)
	cw.count += n
	return n, err
}

func runShellCommand(ctx context.Context, shellCmd []string) (*ShellResult, error) {
	var stdoutWriter *_CountingWriter
	var stdErrWriter *_CountingWriter

	// Execute the shell command
	// Note: This is a blocking call
	//nolint:gosec  // User is deliberately executing a command
	cmdCtx := exec.CommandContext(ctx, shellCmd[0], shellCmd[1:]...)
	if isInteractiveTerminal() {
		log.Debug().Msg("Running shell command in interactive mode")
		stdoutWriter = newCountingWriter(os.Stdout)
		stdErrWriter = newCountingWriter(os.Stderr)
		cmdCtx.Stdin = os.Stdin
		cmdCtx.Stdout = stdoutWriter
		cmdCtx.Stderr = stdErrWriter
	}
	// cmdCtx.Stdout = log.Logger.Level(zerolog.InfoLevel).With().Logger()
	// cmdCtx.Stderr = log.Logger.Level(zerolog.ErrorLevel).With().Strs("shellCmd", shellCmd).Logger()
	err := cmdCtx.Run()

	var shellResult *ShellResult
	if stdoutWriter != nil && stdErrWriter != nil {
		shellResult = &ShellResult{stdoutBytes: stdoutWriter.count, stderrBytes: stdErrWriter.count}
	}

	// Check for other errors and return them as-is
	if err != nil {
		return shellResult, fmt.Errorf("failed to run command: %w", err)
	}

	log.Debug().
		Strs("shellCmd", shellCmd).
		Msg("Command ran successfully")
	return shellResult, nil
}
