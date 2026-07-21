# aztx - Azure-CLI Context Switcher

`aztx` is a command-line tool designed to streamline the management of Azure tenant and subscription contexts. It provides an intuitive fuzzy-finder interface for switching between Azure subscriptions and tenants, making it easier to work with multiple Azure environments.

> [!NOTE]
> This is a fork of [riweston/aztx](https://github.com/riweston/aztx) that makes **per-shell isolation** the default mode of operation: switching context never mutates `~/.azure`.

## Features

- 🐚 **Per-shell isolated contexts** — each `aztx` invocation copies `~/.azure` to a private tempdir, sets `AZURE_CONFIG_DIR`, and drops you into a subshell; the master `~/.azure` is never touched
- 🔍 Fuzzy search interface for finding subscriptions and tenants
- ⚡ Quick context switching between subscriptions
- 🔄 Easy switching to previous context (similar to `cd -`)
- 🎯 Tenant-first selection mode
- 🔧 Configurable logging levels

### Demo

[![asciicast](https://asciinema.org/a/Rk36acdIGN9K6w5WO5Rx74NwA.svg)](https://asciinema.org/a/Rk36acdIGN9K6w5WO5Rx74NwA)

## Prerequisites

> [!NOTE]
> This tool is built on top of the azure-cli and fzf and requires them to be installed and configured.
> If you use the Brew or Scoop package managers, these pre-requisites will be handled during installation.

- go >=1.16.6
- azure-cli >= 2.22.1
- fzf >= 0.20.0

## Installation

### [Brew](https://brew.sh/) (Mac/Linux)

```sh
brew tap riweston/aztx
brew install aztx
```

### [Scoop](https://scoop.sh/) (Windows)

```sh
scoop bucket add riweston https://github.com/riweston/scoop-bucket.git
scoop update
scoop install riweston/aztx
```

### [winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/) (Windows)

```sh
winget install aztx
```

### Download Prebuilt Binary

Download the latest release from the [releases page](https://github.com/riweston/aztx/releases) and add it to your PATH.

### Install from Source

```sh
go install github.com/riweston/aztx
```

## Usage

### Basic Subscription Switching

```sh
# Pick a subscription and drop into a subshell scoped to it.
# ~/.azure is copied to a tempdir and AZURE_CONFIG_DIR points at the copy,
# so the pick never mutates your master config.
aztx

# Inside the isolated shell:
echo $AZURE_CONFIG_DIR   # /tmp/aztx.XXXXXXX
az account show          # shows the picked subscription

# An isolated shell is bound to its subscription for its whole lifetime:
# re-running aztx inside one is refused (it couldn't update the shell's
# AZTX_SUBSCRIPTION). Exit the shell and re-run aztx, or use aztx exec.

# Exit the subshell; the tempdir is cleaned up automatically.
exit

# Switch to previous subscription context
aztx -

# Clear the default subscription in the master ~/.azure (no picker,
# no subshell). Refused inside an isolated shell.
aztx --unset

# List isolated contexts: config dir, subscription, owning PID, age.
# '*' marks the current shell's context.
aztx list

# Show the current shell's context as indented JSON (exit 1 outside
# an isolated shell) — subscription, tenant, PID, env consistency.
aztx status
```

Every aztx invocation garbage-collects orphaned contexts: each tempdir
records its owning process, and dirs whose owner is gone (e.g. after a
SIGKILL) are removed automatically on the next run.

### Tenant-First Selection

```sh
# Select tenant before choosing subscription
aztx --by-tenant
```

### Exec Mode

```sh
# Pick a subscription, run a single command in the isolated context,
# then clean up — similar to aws-vault exec. The command's exit code
# is propagated.
aztx exec -- kubectl get pods
aztx exec --by-tenant -- kubie ctx my-aks-cluster

# Skip the picker entirely with --subscription (name or ID, name is
# case-insensitive). Also works on bare aztx.
aztx exec --subscription "My Subscription" -- kubectl get pods
```

### In-Place Mode

```sh
# Mutate the master ~/.azure directly (upstream behavior): no tempdir
# copy, no subshell. Refused inside an isolated shell like bare aztx.
aztx --in-place
```

Notes on isolation:

- The subshell is `$SHELL` (fallback `/bin/zsh`).
- The spawned shell/command gets `AZTX_SUBSCRIPTION` set to the picked
  subscription's name (like aws-vault's `AWS_VAULT`), handy for prompts
  and wrapper scripts. It is always accurate because a shell's
  subscription is immutable — re-picking inside an isolated shell is
  refused, with no override.
- Each isolated context gets its own copy of the token cache. If tokens
  expire, `az login` inside the subshell only affects that context.
- `kubelogin`/`kubectl` honor `AZURE_CONFIG_DIR`, so AKS access works
  inside the isolated shell.

## Configuration

Configuration is stored in `~/.aztx.yml`. The following options are available:

```yaml
# Log level: debug, info, warn, error
log-level: info

# by-tenant: true, false
by-tenant: false
```

You can also set configuration via environment variables:
- `AZTX_LOG_LEVEL`: Set logging level
- `AZTX_BY_TENANT`: Enable tenant-first selection mode

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Show your support

Give a ⭐️ if this project helped you!
