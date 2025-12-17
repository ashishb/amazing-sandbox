# Amazing Sandbox (AS)

Amazing Sandbox (AS) is for running various tools inside a Docker sandbox in a brain-dead fashion.

- [x] Prevents [malicious packages](https://www.kaspersky.com/about/press-releases/kaspersky-uncovers-500k-crypto-heist-through-malicious-packages-targeting-cursor-developers) from having full disk access and stealing data
- [x] Prevents AI agents from [mistakenly](https://www.theregister.com/2025/12/01/google_antigravity_wipes_d_drive/) deleting all files on your disk
- [x] Optionally, run packages like linters with no internet access as well

## Features

Default config

- [x] Give Read-write access to current directory
- [x] network access
- [x] Load `.env` file from the current directory
- [x] Cache various build steps using Docker
- [x] Give Read-write access to any explictly referenced files via CLI arguments

Planned via CLI config

- [ ] Disable Read-write access to current directory
- [ ] Give Read-only access to current directory
- [ ] Disable network access
- [ ] Disable `.env` file loading
- [ ] Disable Read-write access to any explictly referenced files via CLI arguments

## Supported

- [x] `npx`
- [x] `npm`
- [x] `yarn`
- [x] `cargo` and `cargo-exec`
- [x] `gem` and `gem-exec`
- [ ] `uv`

## Usage

```bash
$ as yarn install   # Run npm with full access to current directory + a cache directory but no access to full disk
$ asn npx htmlhint  # asn = Amazing sandbox (no Internet) access

# To see the full usage
$ asn --help
...
```
