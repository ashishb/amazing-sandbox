# Amazing Sandbox (AS)

Amazing Sandbox (AS) is for running various tools inside a Docker sandbox in a brain-dead fashion.

[x] Prevents [malicious packages](https://www.kaspersky.com/about/press-releases/kaspersky-uncovers-500k-crypto-heist-through-malicious-packages-targeting-cursor-developers) from having full disk access and stealing data
[x] Prevents AI agents from [mistakenly](https://www.theregister.com/2025/12/01/google_antigravity_wipes_d_drive/) deleting all files on your disk
[x] Optionally, run packages like linters with no internet access as well

## Features

- [x] Give Read-write access to current directory (enabled by default)
- [x] Give Read-only access to current directory
- [x] Give network access (enabled by default)
- [x] Load `.env` file from the current directory (enabled by default)
- [x] Cache various build steps using Docker
- [x] Give Read-write access to any explictly referenced files via CLI arguments (enabled by default)

## Supported

- [x] `npx`
- [x] `npm`
- [x] `yarn`
- [ ] `cargo`
- [ ] `brew`
- [x] `gem` and `gem-exec`

## Usage

```bash
$ as yarn install   # Run npm with full access to current directory + a cache directory but no access to full disk
$ asn npx htmlhint  # asn = Amazing sandbox (no Internet) access

# To see the full usage
$ asn --help
...
