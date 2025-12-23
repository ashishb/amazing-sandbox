# Amazing Sandbox (`asb`)

[![Lint GitHub Actions](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-github-actions.yaml/badge.svg)](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-github-actions.yaml)
[![Lint Markdown](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-markdown.yaml/badge.svg)](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-markdown.yaml)
[![Lint YAML](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-yaml.yaml/badge.svg)](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-yaml.yaml)

[![Lint Go](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-go.yaml/badge.svg)](https://github.com/ashishb/amazing-sandbox/actions/workflows/lint-go.yaml)
[![Validate Go code formatting](https://github.com/ashishb/amazing-sandbox/actions/workflows/format-go.yaml/badge.svg)](https://github.com/ashishb/amazing-sandbox/actions/workflows/format-go.yaml)

Amazing Sandbox (AS) is for running various tools inside a Docker sandbox.

- [x] Prevents [malicious packages](https://www.kaspersky.com/about/press-releases/kaspersky-uncovers-500k-crypto-heist-through-malicious-packages-targeting-cursor-developers) from having full disk access and stealing data
- [x] Prevents AI agents from [mistakenly](https://www.theregister.com/2025/12/01/google_antigravity_wipes_d_drive/) deleting all files on your disk
- [x] Optionally, run packages like linters [air-gapped](https://en.wikipedia.org/wiki/Air_gap_(networking)) (no internet access) as well

> [!WARNING]
> As of Dec 2025, this package is experimental

## Features

Default config

- [x] Give Read-write access to the current directory
- [x] network access
- [x] Load `.env` file from the current directory
- [x] Cache various build steps using Docker
- [x] Give Read-write access to any explicitly referenced files via CLI arguments

Planned via CLI config

- [ ] Disable Read-write access tothe  current directory
- [ ] Give Read-only access to the current directory
- [x] Disable network access - via `-n`
- [ ] Disable `.env` file loading
- [ ] Disable Read-write access to any explicitly referenced files via CLI arguments

## Supported

- JavaScript/Typescript
   - [x] `npx`
   - [x] `npm`
   - [x] `yarn`
   - [x] `pnpm` - Use `asb npx pnpm`
   - [ ] `bun`
- [x] Rust `cargo` and `cargo-exec`
- [x] Ruby `gem` and `gem-exec`
- Python
   - [ ] `pip`
   - [ ] `poetry`
   - [ ] `uv`
   - [x] `uvx`

### Caches config of the following coding agents

1. Claude code
1. OpenAI Codex
1. Google Gemini CLI

### Installation

```
$ go install github.com/ashishb/amazing-sandbox/src/asb@latest
...
```

Or download a binary from the [releases page](https://github.com/ashishb/amazing-sandbox/releases)

## Usage

## Run [yarn](https://yarnpkg.com/) with full access to current directory + a cache directory but no access to full disk

```bash
$ asb yarn install
...
```

## Run [HTML linter](https://www.npmjs.com/package/htmlhint) inside sandbox with `-n`, that is, no Internet access

```bash
$ asb -n npx htmlhint
...  
```

## Run [yamllint](https://github.com/adrienverge/yamllint) inside the sandbox

```bash
$ asb uvx yamllint -d <path-to-dir-containing-yaml-files-to-lint>
...  
```

## Run [Claude code](https://code.claude.com/docs/en/overview) against the current directory

```bash
$ asb npx @anthropic-ai/claude-code
...  
```

## Run [Open AI Codex](https://openai.com/codex/) against the  directory "~/src/repo1"

```bash
$ asb -d ~/src/repo1 npx @openai/codex
```

### Run [Google Gemini CLI](https://github.com/google-gemini/gemini-cli) inside the sandbox

```bash
$ asb npx @google/gemini-cli@latest
```

## To see the full usage

```bash
$ asb --help
...
```

## FAQ

1. Why not use [bubblewrap](https://github.com/containers/bubblewrap)?
   It only [supports](https://github.com/containers/bubblewrap/issues/396) GNU/Linux.
   Further, the developer experience for trying to run a simple tool like `htmlhint` or `yamllint` is sub-par.
