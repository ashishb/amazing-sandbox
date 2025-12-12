package cmdrunner

type (
	CmdType     string
	NetworkType string
)

// Eventually, more command types will be added here
const (
	CmdTypeNpx CmdType = "npx"
)

// Ref: https://docs.docker.com/engine/network/
const (
	NetworkHost   NetworkType = "host"
	NetworkNone   NetworkType = "none"
	NetworkBridge NetworkType = "bridge"
)
