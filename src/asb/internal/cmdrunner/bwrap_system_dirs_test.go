package cmdrunner

import (
	"testing"
)

func TestGetBwrapSystemDirMountArgs(t *testing.T) {
	t.Parallel()
	args := getBwrapSystemDirMountArgs()
	if len(args) != 3*len(_bwrapSystemDirs) {
		t.Fatalf("getBwrapSystemDirMountArgs() returned %d args, want %d", len(args), 3*len(_bwrapSystemDirs))
	}

	bindOptions := make(map[string]string)
	for i := 0; i < len(args); i += 3 {
		// Each directory is mounted at the same path inside the sandbox
		if args[i+1] != args[i+2] {
			t.Errorf("%s mounts %q at %q, want the same path", args[i], args[i+1], args[i+2])
		}
		bindOptions[args[i+1]] = args[i]
	}

	// bwrap aborts if the source of --ro-bind is missing, so directories that don't exist
	// on every distribution must use --ro-bind-try
	for _, dir := range []string{"/lib64", "/snap", "/opt"} {
		if got := bindOptions[dir]; got != "--ro-bind-try" {
			t.Errorf("%s is mounted with %q, want --ro-bind-try", dir, got)
		}
	}
	// The sandbox can't work without these, so fail early if they are missing
	for _, dir := range []string{"/etc", "/usr"} {
		if got := bindOptions[dir]; got != "--ro-bind" {
			t.Errorf("%s is mounted with %q, want --ro-bind", dir, got)
		}
	}
}
