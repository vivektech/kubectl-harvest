# Installing kubectl-harvest

`kubectl-harvest` ships as a single static binary for every platform, published on the [GitHub Releases](https://github.com/vivektech/kubectl-harvest/releases) page:

| OS | Architectures | Archive |
|---|---|---|
| macOS (Intel) | amd64 | `.tar.gz` |
| macOS (Apple Silicon M1/M2/M3/M4) | arm64 | `.tar.gz` |
| Linux (x86-64) | amd64 | `.tar.gz` |
| Linux (ARM servers / Raspberry Pi 4+ / Graviton) | arm64 | `.tar.gz` |
| Windows (x64) | amd64 | `.zip` |
| Windows on ARM | arm64 | `.zip` |

Pick any of the four installation methods below. All of them give you the same result: a `kubectl-harvest` binary on your `PATH`, which kubectl automatically picks up as the plugin command `kubectl harvest`.

Verify afterwards with:

```
kubectl harvest --version
```

---

## macOS

### Option 1 — Homebrew (recommended)

Works on both Intel and Apple Silicon Macs; Homebrew picks the right binary automatically.

```console
$ brew tap vivektech/kubectl-harvest https://github.com/vivektech/kubectl-harvest
$ brew install kubectl-harvest
```

To upgrade later:

```console
$ brew upgrade kubectl-harvest
```

### Option 2 — Krew

```console
$ kubectl krew index add vivektech https://github.com/vivektech/kubectl-harvest
$ kubectl krew install vivektech/harvest
```

(If you don't have Krew yet: https://krew.sigs.k8s.io/docs/user-guide/setup/install/)

### Option 3 — Prebuilt binary

Apple Silicon (M1/M2/M3/M4):

```console
$ curl -LO https://github.com/vivektech/kubectl-harvest/releases/download/v1.0.2/kubectl-harvest_1.0.2_darwin_arm64.tar.gz
$ tar -xzf kubectl-harvest_1.0.2_darwin_arm64.tar.gz
$ sudo mv kubectl-harvest /usr/local/bin/
$ kubectl harvest --version
```

Intel:

```console
$ curl -LO https://github.com/vivektech/kubectl-harvest/releases/download/v1.0.2/kubectl-harvest_1.0.2_darwin_amd64.tar.gz
$ tar -xzf kubectl-harvest_1.0.2_darwin_amd64.tar.gz
$ sudo mv kubectl-harvest /usr/local/bin/
$ kubectl harvest --version
```

On Apple Silicon Macs, `/usr/local/bin` may not exist yet — create it first with `sudo mkdir -p /usr/local/bin`, or use `~/.local/bin` (see "Binary install notes" below).

### Option 4 — Go

Requires Go 1.24 or newer:

```console
$ go install github.com/vivektech/kubectl-harvest/cmd/kubectl-harvest@latest
```

Go installs to `$(go env GOPATH)/bin` (usually `~/go/bin`) — make sure it is on your `PATH`.

---

## Linux

### Option 1 — Homebrew on Linux

```console
$ brew tap vivektech/kubectl-harvest https://github.com/vivektech/kubectl-harvest
$ brew install kubectl-harvest
```

### Option 2 — Krew

```console
$ kubectl krew index add vivektech https://github.com/vivektech/kubectl-harvest
$ kubectl krew install vivektech/harvest
```

### Option 3 — Prebuilt binary

x86-64:

```console
$ curl -LO https://github.com/vivektech/kubectl-harvest/releases/download/v1.0.2/kubectl-harvest_1.0.2_linux_amd64.tar.gz
```

ARM64 (Graviton, Ampere, Raspberry Pi 4/5 with a 64-bit OS):

```console
$ curl -LO https://github.com/vivektech/kubectl-harvest/releases/download/v1.0.2/kubectl-harvest_1.0.2_linux_arm64.tar.gz
```

Then, either root-wide or per-user:

```console
$ tar -xzf kubectl-harvest_1.0.2_linux_*.tar.gz

# root-wide
$ sudo mv kubectl-harvest /usr/local/bin/

# or per-user (recommended if you don't have sudo)
$ mkdir -p ~/.local/bin && mv kubectl-harvest ~/.local/bin/
$ echo 'export PATH="$PATH:$HOME/.local/bin"' >> ~/.bashrc   # or ~/.zshrc
$ source ~/.bashrc

$ kubectl harvest --version
```

### Option 4 — Go

```console
$ go install github.com/vivektech/kubectl-harvest/cmd/kubectl-harvest@latest
```

---

## Windows

### Option 1 — Krew

Works in PowerShell or CMD (Krew on Windows requires `kubectl` 1.22+):

```console
> kubectl krew index add vivektech https://github.com/vivektech/kubectl-harvest
> kubectl krew install vivektech/harvest
```

### Option 2 — Prebuilt binary

Download `kubectl-harvest_1.0.2_windows_amd64.zip` (or `..._windows_arm64.zip` on ARM devices) from the [Releases](https://github.com/vivektech/kubectl-harvest/releases) page, extract `kubectl-harvest.exe`, and put it somewhere on your `PATH`.

PowerShell example:

```powershell
PS> Expand-Archive kubectl-harvest_1.0.2_windows_amd64.zip -DestinationPath $env:USERPROFILE\bin
PS> $env:Path += ";$env:USERPROFILE\bin"          # for this session
# persist permanently:
PS> [Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\bin", "User")

PS> kubectl harvest --version
```

### Option 3 — Go

```console
> go install github.com/vivektech/kubectl-harvest/cmd/kubectl-harvest@latest
```

Installs to `%USERPROFILE%\go\bin` — add it to your `PATH` if it isn't already.

### Option 4 — Chocolatey / Scoop

Not yet packaged. Homebrew-style taps above are the maintained channels; a community package is welcome.

---

## Binary install notes (all OSes)

- **The binary must be named `kubectl-harvest`** (on Windows: `kubectl-harvest.exe`) and live on your `PATH` — that is how kubectl discovers plugins and turns it into `kubectl harvest`.
- Check the archive integrity against `checksums.txt` published with each release:
  - macOS/Linux: `shasum -a 256 -c --ignore-missing checksums.txt`
  - Windows: `Get-FileHash .\kubectl-harvest.exe -Algorithm SHA256`
- On macOS, binaries from GitHub Releases are **not notarized**; if Gatekeeper complains, remove the quarantine attribute: `xattr -d com.apple.quarantine $(which kubectl-harvest)`. The Homebrew and Krew channels avoid this entirely.

## Uninstall

- **Homebrew:** `brew uninstall kubectl-harvest && brew untap vivektech/kubectl-harvest`
- **Krew:** `kubectl krew uninstall harvest`
- **Binary/Go:** delete `kubectl-harvest` from wherever you put it
