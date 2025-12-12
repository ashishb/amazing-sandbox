package cmdrunner

import "github.com/rs/zerolog/log"

const (
	_npmDockerImage = "node:25-bookworm-slim"
	_npxDockerImage = _npmDockerImage
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
		switch c.cmdType {
		case CmdTypeNpm:
			c.args = append([]string{"npm"}, args...)
		case CmdTypeNpx:
			c.args = append([]string{"npx"}, args...)
		case CmdTypeRubyGem:
			// Make sure to use --conservative flag for install command
			// to avoid attemping to update already installed gems
			if len(args) > 0 && args[0] == "install" {
				c.args = append([]string{"gem", "install", "--conservative"}, args[1:]...)
			} else {
				c.args = append([]string{"gem"}, args...)
			}
		case CmdTypeRubyGemExec:
			c.args = args
		default:
			log.Fatal().
				Str("cmdType", string(c.cmdType)).
				Msg("Unsupported command type for setting args")
		}
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

func NewNpmCmdConfig(options ...Option) Config {
	cfg := &Config{
		dockerBaseImage:   _npmDockerImage,
		cmdType:           CmdTypeNpm,
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

func NewRubyGemCmdConfig(options ...Option) Config {
	cfg := &Config{
		dockerBaseImage:   "ruby:3-bookworm",
		cmdType:           CmdTypeRubyGem,
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

func NewRubyGemExecCmdConfig(options ...Option) Config {
	cfg := &Config{
		dockerBaseImage:   "ruby:3-bookworm",
		cmdType:           CmdTypeRubyGemExec,
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
