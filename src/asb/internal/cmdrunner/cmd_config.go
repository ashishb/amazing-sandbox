package cmdrunner

const (
	_npxDockerImage = "node:25-bookworm-slim"
)

type Config struct {
	dockerBaseImage string // Docker base image to use
	cmdType         CmdType
	workingDir      string   // Working directory for the command
	args            []string // Optional arguments to the command

	// At most one of these should be true
	mountWorkingDirRW bool // Whether to mount the working directory into the container as read-write
	mountWorkingDirRO bool // Whether to mount the working directory into the container as read-only

	runAsNonRoot bool        // Whether to run the container as non-root user
	networkType  NetworkType // Network type for the container
}

type Option func(*Config)

func SetWorkingDir(workingDir string) Option {
	return func(c *Config) {
		c.workingDir = workingDir
	}
}

func SetArgs(args []string) Option {
	return func(c *Config) {
		c.args = args
	}
}

func SetNetworkType(networkType NetworkType) Option {
	return func(c *Config) {
		c.networkType = networkType
	}
}

func SetRunAsNonRoot(runAsNonRoot bool) Option {
	return func(c *Config) {
		c.runAsNonRoot = runAsNonRoot
	}
}

func SetMountWorkingDirReadOnly(mountRO bool) Option {
	return func(c *Config) {
		if mountRO {
			c.mountWorkingDirRW = false
		}
		c.mountWorkingDirRO = mountRO
	}
}

func SetMountWorkingDirReadWrite(mountRW bool) Option {
	return func(c *Config) {
		if mountRW {
			c.mountWorkingDirRO = false
		}
		c.mountWorkingDirRW = mountRW
	}
}

func NewNpxCmdConfig(options ...Option) Config {
	cfg := &Config{
		dockerBaseImage:   _npxDockerImage,
		cmdType:           CmdTypeNpx,
		workingDir:        ".",
		args:              nil,
		mountWorkingDirRW: true,
		mountWorkingDirRO: false,
		runAsNonRoot:      true,
		networkType:       NetworkHost,
	}

	for _, option := range options {
		option(cfg)
	}

	return *cfg
}
