package cmdrunner

import (
	"os"
	"path"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	_uvDockerImage     = "astral/uv:python3.12-bookworm-slim"
	_pipDockerImage    = _uvDockerImage
	_poetryDockerImage = _uvDockerImage

	_rustCargoDockerImage = "rust:1.92"
	_rubyDockerImage      = "ruby:3-bookworm"
	_haskellDockerImage   = "haskell:9.10"
	_goDockerImage        = "golang:1.26"
	_zigDockerImage       = "kassany/bookworm-ziglang:0.16.0"

	// Note that node:25-bookworm-slim does not contain C/C++ build tools and that makes anything
	// using node-gyp to fail. Hence we use the full image here.
	_npmDockerImage  = "node:25-bookworm"
	_nodeDockerImage = _npmDockerImage
	_yarnDockerImage = _npmDockerImage
	_npxDockerImage  = _npmDockerImage
	_pnpmDockerImage = _npmDockerImage
	_bunDockerImage  = "oven/bun:debian"
)

var _dockerImageMap = map[CmdType]string{
	CmdTypeBun:              _bunDockerImage,
	CmdTypeNode:             _nodeDockerImage,
	CmdTypeNpm:              _npmDockerImage,
	CmdTypeYarn:             _yarnDockerImage,
	CmdTypeRustCargo:        _rustCargoDockerImage,
	CmdTypeRustCargoExec:    _rustCargoDockerImage,
	CmdTypePythonPip:        _pipDockerImage,
	CmdTypePythonPipExec:    _pipDockerImage,
	CmdTypePython:           _uvDockerImage,
	CmdTypePythonUv:         _uvDockerImage,
	CmdTypePythonUvx:        _uvDockerImage,
	CmdTypePythonPoetry:     _poetryDockerImage,
	CmdTypeNpx:              _npxDockerImage,
	CmdTypePnpm:             _pnpmDockerImage,
	CmdTypeRubyGem:          _rubyDockerImage,
	CmdTypeRubyGemExec:      _rubyDockerImage,
	CmdTypeHaskellCabal:     _haskellDockerImage,
	CmdTypeHaskellCabalExec: _haskellDockerImage,
	CmdTypeGoExec:           _goDockerImage,
	CmdTypeZig:              _zigDockerImage,
}

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

	extraMountRODirs []string // Additional directories to mount as read-only inside the container

	runAsNonRoot bool        // Whether to run the container as non-root user
	networkType  NetworkType // Network type for the container
	loadDotEnv   bool        // Whether to load .env file from working directory
	// Native mode uses sandbox-exec on macOS and bubblewrap (bwrap) on Linux
	execMode ExecMode // Execution mode (docker or native)
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
		// TODO: eliminate special case
		if c.cmdType == CmdTypePnpm && c.args[0] == "npx" {
			c.cmdType = CmdTypeNpx
		}
	}
}

func SetNetworkType(networkType NetworkType) Option {
	return func(c *Config) {
		c.networkType = networkType
	}
}

func SetCustomDockerImage(dockerImage string) Option {
	return func(c *Config) {
		if dockerImage != "" {
			c.dockerBaseImage = dockerImage
		}
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

func SetExtraMountRODirs(dirs []string) Option {
	return func(c *Config) {
		c.extraMountRODirs = append([]string(nil), dirs...)
	}
}

func SetExecMode(execMode ExecMode) Option {
	return func(c *Config) {
		switch execMode {
		case ExecModeDocker:
			c.execMode = execMode
			// Docker is the default mode, so we don't need to do anything here
		case ExecModeNative:
			c.execMode = execMode
		default:
			log.Fatal().
				Str("execMode", string(execMode)).
				Msg("Unsupported execution mode")
		}
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

func (c Config) getDirsToMount() []_FilePathToMount {
	result := make([]_FilePathToMount, 0)
	if c.mountWorkingDirRW {
		result = append(result, newFilePathToMount(c.workingDir, c.workingDir, false))
	} else if c.mountWorkingDirRO {
		result = append(result, newFilePathToMount(c.workingDir, c.workingDir, true))
	}

	if c.getReferencedFiles() != nil {
		for _, dir := range c.getReferencedFiles() {
			result = append(result, newFilePathToMount(dir, dir, c.mountReferencedDirRO))
		}
	}

	for _, dir := range c.extraMountRODirs {
		absDir := getAbsolutePath(c.workingDir, dir)
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			log.Warn().
				Str("dir", absDir).
				Msg("Extra read-only mount directory does not exist, skipping")
			continue
		}
		result = append(result, newFilePathToMount(absDir, absDir, true))
	}

	return result
}

// GetCwdOrFail returns the current working directory, aborting the process if it
// cannot be determined.
func GetCwdOrFail() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Error getting current working directory")
	}
	return cwd
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
	dockerImage, ok := _dockerImageMap[cmdType]
	if !ok {
		log.Fatal().
			Str("cmdType", string(cmdType)).
			Msg("Unsupported command type for getting docker image")
		return ""
	}
	return dockerImage
}

func (cmdType CmdType) getArgs(args []string) []string {
	cmdNameMapping := map[CmdType]string{
		// Rust related
		CmdTypeRustCargo: "cargo",
		// Javascript related
		CmdTypeBun:  "bun",
		CmdTypeNode: "node",
		CmdTypeNpm:  "npm",
		CmdTypeNpx:  "npx",
		CmdTypePnpm: "npx --yes pnpm", // npx handles downloading and caching pnpm automatically
		CmdTypeYarn: "yarn",
		// Python related
		CmdTypePythonPip:    "pip",
		CmdTypePython:       "python",
		CmdTypePythonUv:     "uv",
		CmdTypePythonUvx:    "uvx",
		CmdTypePythonPoetry: "uvx poetry",
		// CmdTypeRubyGem is handled separately below
		CmdTypePythonPipExec: "",
		CmdTypeRubyGemExec:   "gem exec",
		CmdTypeRustCargoExec: "",
		// Haskell related
		CmdTypeHaskellCabal:     "cabal",
		CmdTypeHaskellCabalExec: "",
		// Go related
		CmdTypeGoExec: "go run",
		// Zig related
		CmdTypeZig: "zig",
	}

	if cmdType == CmdTypeRubyGem {
		// Make sure to use --conservative flag for install & exec command
		// to avoid attempting to update already installed gems
		if len(args) > 0 && args[0] == "install" && !slices.Contains(args, "--conservative") {
			return append([]string{"gem", "install", "--conservative"}, args[1:]...)
		}
		if len(args) > 0 && args[0] == "exec" && !slices.Contains(args, "--conservative") {
			return append([]string{"gem", "exec", "--conservative"}, args[1:]...)
		}

		return append([]string{"gem"}, args...)
	}

	if cmdType == CmdTypeGoExec {
		// Remove http:// and https:// from the beginning of the first arg if it exists, because "go run" does not
		// support them, and it can be common for users to include them when copy-pasting
		if len(args) > 0 {
			firstArg := args[0]
			firstArg = strings.TrimPrefix(firstArg, "http://")
			firstArg = strings.TrimPrefix(firstArg, "https://")
			args[0] = firstArg
			return append([]string{"go", "run"}, args...)
		}
	}

	if cmdName, ok := cmdNameMapping[cmdType]; ok {
		if cmdName == "" {
			return args
		}
		return append(strings.Split(cmdName, " "), args...)
	}

	log.Fatal().
		Str("cmdType", string(cmdType)).
		Msg("Unsupported command type for setting args")
	return args
}
