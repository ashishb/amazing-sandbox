package cmdrunner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestSelectContainerRuntime(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		available []string
		want      ExecMode
	}{
		{"both", []string{"docker", "podman"}, ExecModeDocker},
		{"podman only", []string{"podman"}, ExecModePodman},
		{"docker only", []string{"docker"}, ExecModeDocker},
		{"neither", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := selectContainerRuntime(func(name string) (string, error) {
				if slices.Contains(tc.available, name) {
					return "/bin/" + name, nil
				}
				return "", exec.ErrNotFound
			})
			if got != tc.want || (err != nil) != (tc.want == "") {
				t.Fatalf("selection = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestContainerRunCmd(t *testing.T) {
	t.Parallel()
	for _, mode := range []ExecMode{ExecModeDocker, ExecModePodman} {
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()
			cfg := NewConfig(CmdTypePython, SetWorkingDir("/workspace"), SetNetworkType(NetworkNone), SetLoadDotEnv(true))
			cmd, err := getContainerRunCmd(cfg, mode)
			if err != nil {
				t.Fatal(err)
			}
			if cmd[0] != string(mode) || cmd[len(cmd)-1] != cfg.dockerBaseImage {
				t.Fatalf("unexpected command: %v", cmd)
			}
			if mode == ExecModePodman && !slices.Contains(cmd, "--pull=missing") {
				t.Error("missing Podman image pull policy")
			}
			for _, arg := range []string{"run", "--rm", "--init", "--network=none", "--workdir=/workspace", "--env-file=" + filepath.Join("/workspace", ".env"), "--mount=type=volume,src=uv1,target=/root/.cache/uv/"} {
				if !slices.Contains(cmd, arg) {
					t.Errorf("missing %q in %v", arg, cmd)
				}
			}
		})
	}
}

func TestQualifyPodmanImage(t *testing.T) {
	t.Parallel()
	for input, want := range map[string]string{
		"debian:bookworm":       "docker.io/library/debian:bookworm",
		"astral/uv:latest":      "docker.io/astral/uv:latest",
		"ghcr.io/org/image:tag": "ghcr.io/org/image:tag",
		"localhost:5000/image":  "localhost:5000/image",
		"localhost/image":       "localhost/image",
	} {
		if got := qualifyPodmanImage(input); got != want {
			t.Errorf("qualifyPodmanImage(%q) = %q, want %q", input, got, want)
		}
	}
}

// Fake CLIs verify the entire default execution path without a running engine.
func TestRunCmdFallsBackToPodman(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake CLI uses a POSIX shell")
	}
	dir := t.TempDir()
	output := filepath.Join(dir, "args")
	t.Setenv("PATH", dir)
	t.Setenv("ASB_TEST_ARGS", output)
	for name, script := range map[string]string{
		"podman": "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$ASB_TEST_ARGS\"\nexit 42\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfg := NewConfig(CmdTypePython, SetWorkingDir(dir), SetArgs([]string{"--version"}))
	_, err := RunCmd(context.Background(), cfg)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 42 {
		t.Fatalf("expected Podman exit code 42, got %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "docker.io/astral/uv:python3.12-bookworm-slim\npython\n--version\n") {
		t.Fatalf("unexpected Podman arguments: %s", data)
	}

	cfg = NewConfig(CmdTypePython, SetWorkingDir(dir), SetCustomDockerImage("localhost/custom:latest"), SetExecMode(ExecModePodman))
	_, err = RunCmd(context.Background(), cfg)
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 42 {
		t.Fatalf("expected Podman exit code 42, got %v", err)
	}
	data, err = os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "localhost/custom:latest\n") {
		t.Fatalf("custom image was not preserved: %s", data)
	}
}

func TestExecModes(t *testing.T) {
	t.Parallel()
	if got := NewConfig(CmdTypePython).execMode; got != ExecModeAuto {
		t.Fatalf("default mode = %q", got)
	}
	for _, mode := range []ExecMode{ExecModeAuto, ExecModePodman, ExecModeDocker, ExecModeNative} {
		if got := NewConfig(CmdTypePython, SetExecMode(mode)).execMode; got != mode {
			t.Errorf("mode = %q, want %q", got, mode)
		}
	}
}
