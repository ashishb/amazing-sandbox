# Amazing Sandbox (`asb`) ![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/ashishb/amazing-sandbox/total)

[![Lint GitHub Actions](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-github-actions.yaml/badge.svg)](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-github-actions.yaml)
[![Lint Markdown](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-markdown.yaml/badge.svg)](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-markdown.yaml)
[![Lint YAML](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-yaml.yaml/badge.svg)](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-yaml.yaml)

[![Lint Go](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-go.yaml/badge.svg)](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-go.yaml)
[![Validate Go code formatting](https://github.com/ashishb/amazing-sandbox/actions/workflows/format-go.yaml/badge.svg)](https://github.com/ashishb/amazing-sandbox/actions/workflows/format-go.yaml)

Amazing Sandbox (AS) is for running various tools inside a [Docker](https://ashishb.net/programming/docker-101/)-based sandbox (by default) or inside
[seatbelt-based](https://igorstechnoclub.com/sandbox-exec/)
 sandbox on Mac OS.

- [x] Prevents [malicious packages](https://www.kaspersky.com/about/press-releases/kaspersky-uncovers-500k-crypto-heist-through-malicious-packages-targeting-cursor-developers) from having full disk access and stealing data
- [x] Prevents AI agents from [mistakenly](https://www.theregister.com/2025/12/01/google_antigravity_wipes_d_drive/) deleting all files on your disk
- [x] Optionally, run packages like linters [air-gapped](https://en.wikipedia.org/wiki/Air_gap_(networking)) (no internet access) as well

## Features

Default config

- [x] Give Read-write access to the current directory
- [x] network access
- [x] Load `.env` file from the current directory
- [x] Cache various build steps using Docker
- [x] Give Read-write access to any explicitly referenced files via CLI arguments

Configurable via CLI parameters

- [x] Disable read access to the current and referenced directories via `-x`
- [x] Provide Read-only access to the referenced directories via `-r`
- [x] Disable network access - via `-n`
- [x] Disable `.env` file loading via `--load-env=false`
- [x] Add ability to pass a custom Docker image via `-i`

## Supported

- Python
   - [x] `pip`
   - [x] `poetry`
   - [x] `uv`
   - [x] `uvx`
- JavaScript/Typescript
   - [x] `npx`
   - [x] `npm`
   - [x] `yarn`
   - [x] `pnpm`
   - [x] `bun`
- [x] Go `go-exec`
- [x] Rust `cargo` and `cargo-exec`
- [x] Ruby `gem` and `gem-exec`
- [x] Haskell `cabal` and `cabal-exec`

### Caches config of the following coding agents

The configuration of the following coding agents is mapped to the corresponding directories in
your home directory, so they will work seamlessly inside the sandbox without needing to
re-authenticate or re-configure them.

1. [Claude code](https://code.claude.com/docs/en/overview)
1. [OpenAI Codex](https://openai.com/codex/)
1. [Google Gemini CLI](https://github.com/google-gemini/gemini-cli)

### Installation

```
$ go install github.com/ashishb/amazing-sandbox/src/asb/cmd/asb@latest
...
```

Or download a binary from the [releases page](https://github.com/ashishb/amazing-sandbox/releases)

## Usage

### Run [yarn](https://yarnpkg.com/) with full access to current directory + a cache directory but no access to full disk

```bash
$ asb yarn install
...
```

### Run [HTML linter](https://www.npmjs.com/package/htmlhint) inside the sandbox with `-n`, that is, no Internet access

```bash
$ asb -n npx htmlhint
...  
```

### Run [yamllint](https://github.com/adrienverge/yamllint) inside the sandbox

```bash
$ asb uvx yamllint -d <path-to-dir-containing-yaml-files-to-lint>
...  
```

### Run [Claude code](https://code.claude.com/docs/en/overview) against the current directory

```bash
$ asb npx @anthropic-ai/claude-code
...  
```

### Run [Open AI Codex](https://openai.com/codex/) against the  directory "~/src/repo1"

```bash
$ asb -d ~/src/repo1 npx @openai/codex
...
```

### Run [Google Gemini CLI](https://github.com/google-gemini/gemini-cli) inside the sandbox

```bash
$ asb npx @google/gemini-cli@latest
...
```

### Run [fd tool](https://github.com/sharkdp/fd) inside the sandbox with no Internet access

```bash
$ asb cargo install fd-find  # One time install
...
$ asb  -n cargo-exec fd '.*.go'
...
```

### Run [hadolint](https://github.com/hadolint/hadolint) (Haskell-based Dockerfile linter) inside the sandbox

```bash
$ asb cabal update && asb cabal install hadolint  # One time install
...
$ asb -n cabal-exec hadolint Dockerfile
...
```

## To see the full usage

```bash
asb is CLI tool for running tools inside Sandbox
See https://ashishb.net/programming/amazing-sandbox/ for reasoning behind this tool

$ asb --help
Usage:
  asb [flags]
  asb [command]

Available Commands:
  bun         Run a bun command
  cabal       Run a Haskell cabal command
  cabal-exec  Run a Haskell-based binary already installed inside sandbox
  cargo       Run a cargo command
  cargo-exec  Run a Rust-based binary package already installed inside sandbox
  completion  Generate the autocompletion script for the specified shell
  gem         Run a Ruby gem-based CLI tool
  go-exec     Run a Go-based binary package using go run
  help        Help about any command
  node        Run a node command
  npm         Run an npm command
  npx         Run an npx command
  pip         Install Python packages using pip
  pip-exec    Run a Python-based package already installed inside sandbox
  pnpm        Run a pnpm command
  poetry      Run a poetry command
  uv          Run a uv command
  uvx         Run a Python-based package already installed inside sandbox using uvx
  version     Display asb version
  yarn        Run a yarn command

Flags:
  -i, --custom-docker-image string   Use a custom Docker image for the sandbox
  -d, --directory string             Working directory for this command (default "<current directory>")
  -h, --help                         help for asb
  -e, --load-env                     Load .env file from working directory (default true)
      --mode string                  Sandbox mode to use (docker or native) (default "docker")
  -m, --mount-ro stringArray         Mount a directory as read-only inside the sandbox (can be specified multiple times)
  -x, --no-disk-access               Disable disk access inside the sandbox
  -n, --no-network                   Disable network access inside the sandbox
  -r, --read-only                    Load working directory and referenced directories as read-only
  -w, --read-write                   Load working directory and referenced directories as read-write (default true)

```

## How I use it

For interactive shells, one can use bash aliases, for example, `alias htmlhint=asb -n npx htmlhint`.
However, this does not work for non-interactive shells, for example, inside [Makefile](https://ashishb.net/programming/use-makefile-for-android/).
So, I prefer creating `~/.local/bin` which contains `htmlhint` [file](https://github.com/ashishb/dotfiles/blob/master/_local_bin/htmlhint)
containing `asb npx htmlhint "$@"` and add `.local/bin` to the `$PATH` in `~/.bash_profile` via `export PATH=$PATH:$HOME/.local/bin`.

## FAQ

1. Why not use [bubblewrap](https://github.com/containers/bubblewrap)?  
   It only [supports](https://github.com/containers/bubblewrap/issues/396) GNU/Linux.  
   Further, the developer experience for trying to run a simple tool like `htmlhint` or `yamllint` is sub-par.
1. Why not use [Firejail](https://github.com/netblue30/firejail)?  
   No support for Mac OS or Windows.  
   Further, the developer experience for trying to run a simple tool like `htmlhint` or `yamllint` is sub-par.
1. Why not use `sandbox-exec` on Mac OS?  
   `sandbox-exec` is [deprecated](https://github.com/openai/codex/issues/215)
   but if you want you can use `asb` to use `sandbox-exec` via `asb --mode=sandbox-exec)
1. Why not use [ai-jail](https://github.com/akitaonrails/ai-jail)?
   `ai-jail` uses OS-level sandboxing via `bwrap` on Linux and the deprecated `sandbox-exec` on macOS.  
   It has no Windows support.  
   In contrast, `asb` uses Docker, which works consistently across Linux, macOS, and Windows.
1. Why not use [drop](https://github.com/wrr/drop)?  
   `drop` uses Linux mount namespaces for sandboxing and only supports Linux.  
   In contrast, `asb` uses Docker, which works consistently across Linux, macOS, and Windows.
1. I heard that Docker is not a [security boundary](https://kayssel.substack.com/p/docker-escape-breaking-out-of-containers)?
   Containers aren't as strong a security boundary as VMs; however, this means that a successful attack now requires infection of the container AND a concurrent container-escape vulnerability.
   That's a really high bar; someone would need to burn a 0-day on that. Taken from [here](https://news.ycombinator.com/item?id=47612726)
