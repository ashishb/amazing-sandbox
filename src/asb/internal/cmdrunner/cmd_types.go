package cmdrunner

type (
	CmdType     string
	NetworkType string
)

// Eventually, more command types will be added here
const (
	CmdTypeNpm         CmdType = "npm"
	CmdTypeYarn        CmdType = "yarn"
	CmdTypeNpx         CmdType = "npx"
	CmdTypeRubyGem     CmdType = "ruby_gem"
	CmdTypeRubyGemExec CmdType = "ruby_gem_exec"
)

// Ref: https://docs.docker.com/engine/network/
const (
	NetworkHost   NetworkType = "host"
	NetworkNone   NetworkType = "none"
	NetworkBridge NetworkType = "bridge"
)
