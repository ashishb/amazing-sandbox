package cmdrunner

import (
	"os"
	"path"

	"github.com/rs/zerolog/log"
)

const (
	_uvDockerImage     = "astral/uv:python3.12-bookworm-slim"
	_pipDockerImage    = _uvDockerImage
	_poetryDockerImage = _uvDockerImage

	_rustCargoDockerImage = "rust:1.92"
	_rubyDockerImage      = "ruby:3-bookworm"

	// Note that node:25-bookworm-slim does not contain C/C++ build tools and that makes anything
	// using node-gyp to fail. Hence we use the full image here.
	_npmDockerImage  = "node:25-bookworm"
	_yarnDockerImage = _npmDockerImage
	_npxDockerImage  = _npmDockerImage
)

type Config struct {
	dockerBaseImage string // Docker base image to use
	cmdType         CmdType
	workingDir      string   // Working directory for the command
	args            []string // Optional arguments to the command

	// At most one of these should be true
	mountWorkingDirRW bool // Whether to mount the working directory into the container as read-write
	mountWorkingDirRO bool // Whether to mount the working directory into the container as read-only

	mountReferencedDirRO bool // Whether to mount the referenced directory into the container as read-only
	mountReferencedDirRW bool // Whether to mount the referenced directory into the container as read-write

	runAsNonRoot bool        // Whether to run the container as non-root user
	networkType  NetworkType // Network type for the container
	loadDotEnv   bool        // Whether to load .env file from working directory
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
		case CmdTypeRustCargo:
			c.args = append([]string{"cargo"}, args...)
		case CmdTypePythonPip:
			c.args = append([]string{"pip"}, args...)
		case CmdTypeNpm:
			c.args = append([]string{"npm"}, args...)
		case CmdTypeNpx:
			c.args = append([]string{"npx"}, args...)
		case CmdTypePythonUvx:
			c.args = append([]string{"uvx"}, args...)
		case CmdTypePythonPoetry:
			c.args = append([]string{"uvx", "poetry"}, args...)
		case CmdTypeYarn:
			c.args = append([]string{"yarn"}, args...)
		case CmdTypeRubyGem:
			// Make sure to use --conservative flag for install command
			// to avoid attemping to update already installed gems
			if len(args) > 0 && args[0] == "install" {
				c.args = append([]string{"gem", "install", "--conservative"}, args[1:]...)
			} else {
				c.args = append([]string{"gem"}, args...)
			}
		case CmdTypeRubyGemExec, CmdTypeRustCargoExec, CmdTypePythonPipExec:
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

func SetMountReferencedDirReadWrite(mountRW bool) Option {
	return func(c *Config) {
		if mountRW {
			c.mountReferencedDirRO = false
		}
		c.mountReferencedDirRW = mountRW
	}
}

func SetLoadDotEnv(loadDotEnv bool) Option {
	return func(c *Config) {
		c.loadDotEnv = loadDotEnv
	}
}

func (c Config) getReferencedFiles() []string {
	// Go through args and find any referenced files/directories
	// For simplicity, we assume any arg that begins with "/" or ".." is a reference to a file/directory
	var dirs []string
	for _, arg := range c.args {
		// Note: This is a simplistic check, in real-world scenarios,
		// you might want to use filepath.IsAbs and also check if the path exists
		if len(arg) > 0 && (arg[0] == '/' || (len(arg) > 1 && arg[0:2] == "..")) {
			file1 := getAbsolutePath(c.workingDir, arg)
			if file1 == c.workingDir {
				log.Debug().
					Msg("Skipping working directory from referenced files to avoid double mount")
				continue
			}
			if _, err := os.Stat(file1); os.IsNotExist(err) {
				log.Debug().
					Str("file", file1).
					Msg("Referenced file/directory does not exist, skipping mount")
				continue
			}

			dirs = append(dirs, file1)
		}
	}
	return dirs
}

func getAbsolutePath(baseDir string, relativeDir string) string {
	if relativeDir[0] == os.PathSeparator {
		return relativeDir
	}

	return path.Clean(baseDir + string(os.PathSeparator) + relativeDir)
}

func getDefaultConfig() Config {
	return Config{
		workingDir:        ".",
		args:              nil,
		mountWorkingDirRW: true,
		mountWorkingDirRO: false,
		runAsNonRoot:      true,
		networkType:       NetworkHost,
	}
}

func NewNpmCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _npmDockerImage
	cfg.cmdType = CmdTypeNpm
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func NewYarnCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _yarnDockerImage
	cfg.cmdType = CmdTypeYarn
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func NewCargoCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _rustCargoDockerImage
	cfg.cmdType = CmdTypeRustCargo
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func NewPipCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _pipDockerImage
	cfg.cmdType = CmdTypePythonPip
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func NewPipExecCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _pipDockerImage
	cfg.cmdType = CmdTypePythonPipExec
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func NewUvxCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _uvDockerImage
	cfg.cmdType = CmdTypePythonUvx
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func NewPoetryCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _poetryDockerImage
	cfg.cmdType = CmdTypePythonPoetry
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func NewNpxCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _npxDockerImage
	cfg.cmdType = CmdTypeNpx
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func NewRubyGemCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _rubyDockerImage
	cfg.cmdType = CmdTypeRubyGem
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func NewRubyGemExecCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _rubyDockerImage
	cfg.cmdType = CmdTypeRubyGemExec
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func NewRustCargoExecCmdConfig(options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = _rustCargoDockerImage
	cfg.cmdType = CmdTypeRustCargoExec
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}
