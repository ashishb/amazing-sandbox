package cmdrunner

import (
	"context"
	"errors"
)

func runCmdInNative(_ context.Context, _ Config) error {
	return errors.New("native execution mode is not supported yet")
}
