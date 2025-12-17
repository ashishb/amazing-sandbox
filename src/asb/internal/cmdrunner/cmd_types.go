package cmdrunner

type (
	CmdType     string
	NetworkType string
)

// Eventually, more command types will be added here
const (
	CmdTypeCargo CmdType = "cargo"

	CmdTypeNpm  CmdType = "npm"
	CmdTypeNpx  CmdType = "npx"
	CmdTypeYarn CmdType = "yarn"

	CmdTypeRubyGem       CmdType = "ruby_gem"
	CmdTypeRubyGemExec   CmdType = "ruby_gem_exec"
	CmdTypeRustCargoExec CmdType = "rust_cargo_exec"
)

// Ref: https://docs.docker.com/engine/network/
const (
	NetworkHost   NetworkType = "host"
	NetworkNone   NetworkType = "none"
	NetworkBridge NetworkType = "bridge"
)
