package cmdrunner

import (
	"os"
	"path"
	"strings"

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
	_bunDockerImage  = "oven/bun:debian"
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
		c.args = c.cmdType.getArgs(args)
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
			c.mountReferencedDirRW = false
		}
		c.mountWorkingDirRO = mountRO
		c.mountReferencedDirRO = mountRO
	}
}

func SetMountWorkingDirReadWrite(mountRW bool) Option {
	return func(c *Config) {
		if mountRW {
			c.mountWorkingDirRO = false
			c.mountReferencedDirRO = false
		}
		c.mountWorkingDirRW = mountRW
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

func NewConfig(cmdType CmdType, options ...Option) Config {
	cfg := getDefaultConfig()
	cfg.dockerBaseImage = cmdType.getDockerImage()
	cfg.cmdType = cmdType
	for _, option := range options {
		option(&cfg)
	}
	return cfg
}

func getDefaultConfig() Config {
	return Config{
		workingDir:           ".",
		args:                 nil,
		mountWorkingDirRW:    true,
		mountWorkingDirRO:    false,
		mountReferencedDirRO: false,
		mountReferencedDirRW: false,
		runAsNonRoot:         true,
		networkType:          NetworkHost,
		loadDotEnv:           false,
	}
}

func (cmdType CmdType) getDockerImage() string {
	switch cmdType {
	case CmdTypeBun:
		return _bunDockerImage
	case CmdTypeNpm:
		return _npmDockerImage
	case CmdTypeYarn:
		return _yarnDockerImage
	case CmdTypeRustCargo, CmdTypeRustCargoExec:
		return _rustCargoDockerImage
	case CmdTypePythonPip, CmdTypePythonPipExec:
		return _pipDockerImage
	case CmdTypePythonUv, CmdTypePythonUvx:
		return _uvDockerImage
	case CmdTypePythonPoetry:
		return _poetryDockerImage
	case CmdTypeNpx:
		return _npxDockerImage
	case CmdTypeRubyGem, CmdTypeRubyGemExec:
		return _rubyDockerImage
	default:
		log.Fatal().
			Str("cmdType", string(cmdType)).
			Msg("Unsupported command type for getting docker image")
		return ""
	}
}

func (cmdType CmdType) getArgs(args []string) []string {
	cmdNameMapping := map[CmdType]string{
		// Rust related
		CmdTypeRustCargo: "cargo",
		// Javascript related
		CmdTypeBun:  "bun",
		CmdTypeNpm:  "npm",
		CmdTypeNpx:  "npx",
		CmdTypeYarn: "yarn",
		// Python related
		CmdTypePythonPip:    "pip",
		CmdTypePythonUv:     "uv",
		CmdTypePythonUvx:    "uvx",
		CmdTypePythonPoetry: "uvx poetry",
		// CmdTypeRubyGem is handled separately below
		CmdTypePythonPipExec: "",
		CmdTypeRubyGemExec:   "",
		CmdTypeRustCargoExec: "",
	}

	if cmdName, ok := cmdNameMapping[cmdType]; ok {
		if cmdName == "" {
			return args
		}
		return append(strings.Split(cmdName, " "), args...)
	}

	if cmdType == CmdTypeRubyGem {
		// Make sure to use --conservative flag for install command
		// to avoid attempting to update already installed gems
		if len(args) > 0 && args[0] == "install" {
			return append([]string{"gem", "install", "--conservative"}, args[1:]...)
		}

		return append([]string{"gem"}, args...)
	}

	log.Fatal().
		Str("cmdType", string(cmdType)).
		Msg("Unsupported command type for setting args")
	return args
}
