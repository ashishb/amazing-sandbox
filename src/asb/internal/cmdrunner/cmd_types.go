package cmdrunner

type (
	CmdType     string
	NetworkType string
)

// Eventually, more command types will be added here
const (
	CmdTypeRustCargo     CmdType = "rust_cargo"
	CmdTypeRustCargoExec CmdType = "rust_cargo_exec"

	CmdTypePythonPip     CmdType = "python_pip"
	CmdTypePythonPipExec CmdType = "python_pip_exec"
	CmdTypePythonUvx     CmdType = "python_uvx"
	CmdTypePythonPoetry  CmdType = "python_poetry"

	CmdTypeBun  CmdType = "bun" // Ref: https://bun.sh/
	CmdTypeNpm  CmdType = "npm"
	CmdTypeNpx  CmdType = "npx"
	CmdTypeYarn CmdType = "yarn"

	CmdTypeRubyGem     CmdType = "ruby_gem"
	CmdTypeRubyGemExec CmdType = "ruby_gem_exec"
)

// Ref: https://docs.docker.com/engine/network/
const (
	NetworkHost   NetworkType = "host"
	NetworkNone   NetworkType = "none"
	NetworkBridge NetworkType = "bridge"
)
