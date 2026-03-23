package cmdrunner

import (
	"testing"
)

func TestNodeDockerImage(t *testing.T) {
	t.Parallel()
	if got := CmdTypeNode.getDockerImage(); got != _nodeDockerImage {
		t.Errorf("CmdTypeNode.getDockerImage() = %q, want %q", got, _nodeDockerImage)
	}
}

func TestNodeArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cmdType  CmdType
		args     []string
		wantArgs []string
	}{
		{
			name:     "node with script prepends node",
			cmdType:  CmdTypeNode,
			args:     []string{"index.js"},
			wantArgs: []string{"node", "index.js"},
		},
		{
			name:     "node with no args",
			cmdType:  CmdTypeNode,
			args:     []string{},
			wantArgs: []string{"node"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.cmdType.getArgs(tt.args)
			if len(got) != len(tt.wantArgs) {
				t.Fatalf("getArgs() = %v, want %v", got, tt.wantArgs)
			}
			for i := range got {
				if got[i] != tt.wantArgs[i] {
					t.Errorf("getArgs()[%d] = %q, want %q", i, got[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestNodeNewConfig(t *testing.T) {
	t.Parallel()
	cfg := NewConfig(CmdTypeNode,
		SetWorkingDir("/tmp"),
		SetArgs([]string{"index.js"}),
		SetNetworkType(NetworkHost),
	)
	if cfg.dockerBaseImage != _nodeDockerImage {
		t.Errorf("dockerBaseImage = %q, want %q", cfg.dockerBaseImage, _nodeDockerImage)
	}
	if cfg.cmdType != CmdTypeNode {
		t.Errorf("cmdType = %q, want %q", cfg.cmdType, CmdTypeNode)
	}
	wantArgs := []string{"node", "index.js"}
	if len(cfg.args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", cfg.args, wantArgs)
	}
	for i := range cfg.args {
		if cfg.args[i] != wantArgs[i] {
			t.Errorf("args[%d] = %q, want %q", i, cfg.args[i], wantArgs[i])
		}
	}
}

func TestPipDockerImage(t *testing.T) {
	t.Parallel()
	if got := CmdTypePythonPip.getDockerImage(); got != _uvDockerImage {
		t.Errorf("CmdTypePythonPip.getDockerImage() = %q, want %q", got, _uvDockerImage)
	}
	if got := CmdTypePythonPipExec.getDockerImage(); got != _uvDockerImage {
		t.Errorf("CmdTypePythonPipExec.getDockerImage() = %q, want %q", got, _uvDockerImage)
	}
}

func TestPipArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cmdType  CmdType
		args     []string
		wantArgs []string
	}{
		{
			name:     "pip install prepends pip",
			cmdType:  CmdTypePythonPip,
			args:     []string{"install", "requests"},
			wantArgs: []string{"pip", "install", "requests"},
		},
		{
			name:     "pip with no args",
			cmdType:  CmdTypePythonPip,
			args:     []string{},
			wantArgs: []string{"pip"},
		},
		{
			name:     "pip-exec passes args through unchanged",
			cmdType:  CmdTypePythonPipExec,
			args:     []string{"mypy", "src/"},
			wantArgs: []string{"mypy", "src/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.cmdType.getArgs(tt.args)
			if len(got) != len(tt.wantArgs) {
				t.Fatalf("getArgs() = %v, want %v", got, tt.wantArgs)
			}
			for i := range got {
				if got[i] != tt.wantArgs[i] {
					t.Errorf("getArgs()[%d] = %q, want %q", i, got[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestPipNewConfig(t *testing.T) {
	t.Parallel()
	cfg := NewConfig(CmdTypePythonPip,
		SetWorkingDir("/tmp"),
		SetArgs([]string{"install", "requests"}),
		SetNetworkType(NetworkHost),
	)
	if cfg.dockerBaseImage != _uvDockerImage {
		t.Errorf("dockerBaseImage = %q, want %q", cfg.dockerBaseImage, _uvDockerImage)
	}
	if cfg.cmdType != CmdTypePythonPip {
		t.Errorf("cmdType = %q, want %q", cfg.cmdType, CmdTypePythonPip)
	}
	wantArgs := []string{"pip", "install", "requests"}
	if len(cfg.args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", cfg.args, wantArgs)
	}
	for i := range cfg.args {
		if cfg.args[i] != wantArgs[i] {
			t.Errorf("args[%d] = %q, want %q", i, cfg.args[i], wantArgs[i])
		}
	}
}
