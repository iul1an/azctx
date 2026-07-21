# azctx - Azure-CLI Context Switcher

`azctx` is a command-line tool designed to streamline the management of Azure tenant and subscription contexts. It provides an intuitive fuzzy-finder interface for switching between Azure subscriptions and tenants, making it easier to work with multiple Azure environments.

> [!NOTE]
> Unlike plain `az account set`, azctx makes **per-shell isolation** the default mode of operation: switching context never mutates `~/.azure`. See [Origins](#origins).

## Features

- 🐚 **Per-shell isolated contexts** — each `azctx` invocation copies `~/.azure` to a private tempdir, sets `AZURE_CONFIG_DIR`, and drops you into a subshell; the master `~/.azure` is never touched
- 🔍 Fuzzy search interface for finding subscriptions and tenants
- ⚡ Quick context switching between subscriptions
- 🔄 Easy switching to previous context (similar to `cd -`)
- 🎯 Tenant-first selection mode
- 🔧 Configurable logging levels

## Prerequisites

- azure-cli >= 2.22.1 (azctx manages its profile; the fuzzy finder itself is built in)
- go >= 1.26 (building from source only)

## Installation

Linux and macOS only — the per-shell isolation relies on Unix process
semantics (`$SHELL`, signals), so there are no Windows builds.

### Download Prebuilt Binary

Download the latest release from the [releases page](https://github.com/iul1an/azctx/releases) and add it to your PATH.

### Install from Source

```sh
go install github.com/iul1an/azctx@latest
# or from a checkout (default /usr/local/bin; override e.g. PREFIX=$HOME for ~/bin):
make install PREFIX=$HOME
```

### Shell Completion

Completions for bash, zsh, fish, and powershell are built in:

```sh
# zsh: write into any directory on your $fpath
azctx completion zsh > ~/.zsh/completions/_azctx

# bash
azctx completion bash > /etc/bash_completion.d/azctx
```

Flag values complete too: `--subscription <Tab>` offers the live
subscription names from your active profile, and `--log-level <Tab>` its
four levels. The generated script delegates to the binary at runtime, so
new subcommands and flags are picked up without regenerating the file.

## Usage

### Basic Subscription Switching

```sh
# Pick a subscription and drop into a subshell scoped to it.
# ~/.azure is copied to a tempdir and AZURE_CONFIG_DIR points at the copy,
# so the pick never mutates your master config.
azctx

# Inside the isolated shell:
echo $AZURE_CONFIG_DIR   # /tmp/azctx.XXXXXXX
az account show          # shows the picked subscription

# An isolated shell is bound to its subscription for its whole lifetime:
# re-running azctx inside one is refused (it couldn't update the shell's
# AZCTX_SUBSCRIPTION). Exit the shell and re-run azctx, or use azctx exec.

# Exit the subshell; the tempdir is cleaned up automatically.
exit

# Switch to previous subscription context
azctx -

# Clear the default subscription in the master ~/.azure (no picker,
# no subshell). Refused inside an isolated shell.
azctx --unset

# Start from a completely empty Azure config: nothing copied from
# ~/.azure, no picker. az behaves as never-logged-in inside; an
# `az login` there vanishes with the shell. Also works with exec.
azctx --fresh
azctx exec --fresh -- az login --use-device-code

# List subscriptions ('*' = default) and isolated contexts (config
# dir, subscription, owning PID, age; '*' = current shell's context).
# --json emits both as indented JSON.
azctx list

# Show the current shell's context as indented JSON (exit 1 outside
# an isolated shell) — subscription, tenant, PID, env consistency.
azctx status
```

Every azctx invocation garbage-collects orphaned contexts: each tempdir
records its owning process, and dirs whose owner is gone (e.g. after a
SIGKILL) are removed automatically on the next run.

### Tenant-First Selection

```sh
# Select tenant before choosing subscription
azctx --by-tenant
```

### Exec Mode

```sh
# Pick a subscription, run a single command in the isolated context,
# then clean up — similar to aws-vault exec. The command's exit code
# is propagated.
azctx exec -- kubectl get pods
azctx exec --by-tenant -- kubie ctx my-aks-cluster

# Skip the picker entirely with --subscription (name or ID, name is
# case-insensitive). Also works on bare azctx.
azctx exec --subscription "My Subscription" -- kubectl get pods
```

Inside an isolated shell, `azctx exec` without `--subscription` inherits
that shell's subscription (via `AZCTX_SUBSCRIPTION`) instead of showing
the picker — the command still runs in its own fresh context.

### In-Place Mode

```sh
# Mutate the master ~/.azure directly: no tempdir
# copy, no subshell. Refused inside an isolated shell like bare azctx.
azctx --in-place
```

Notes on isolation:

- The subshell is `$SHELL` (fallback `/bin/zsh`).
- The spawned shell/command gets `AZCTX_SUBSCRIPTION` set to the picked
  subscription's name (like aws-vault's `AWS_VAULT`), handy for prompts
  and wrapper scripts. It is always accurate because a shell's
  subscription is immutable — re-picking inside an isolated shell is
  refused, with no override.
- Each isolated context gets its own copy of the token cache. If tokens
  expire, `az login` inside the subshell only affects that context.
- `kubelogin`/`kubectl` honor `AZURE_CONFIG_DIR`, so AKS access works
  inside the isolated shell.

## Configuration

Configuration is stored in `~/.azctx.yml`. Every flag can be set there
(precedence: flag > `AZCTX_*` environment variable > config file):

```yaml
# Log level: debug, info, warn, error
log-level: info

# Always pick the tenant before the subscription
by-tenant: false

# Always select this subscription (name or ID) — disables the picker
#subscription: "My Subscription"

# Always start from an empty config (ephemeral-by-default workflow)
#fresh: false

# Careful with these two as persistent settings:
# in-place: true makes bare azctx mutate ~/.azure directly;
# unset: true makes every bare azctx run clear the default and exit.
#in-place: false
#unset: false
```

You can also set configuration via environment variables:
- `AZCTX_LOG_LEVEL`: Set logging level
- `AZCTX_BY_TENANT`: Enable tenant-first selection mode
- `AZCTX_SUBSCRIPTION`: Same as `--subscription`. Note the dual role:
  azctx also *exports* this into isolated shells, which is what makes
  nested `azctx exec` inherit the shell's subscription.

## Origins

azctx began as a fork of [riweston/aztx](https://github.com/riweston/aztx)
by Richard Weston (MIT) and retains its full git history. The fuzzy-finder
picker and the `azureProfile.json` handling descend from that project; the
per-shell isolation model, `exec`/`list`/`status`, `--fresh`, the
subscription-binding semantics, and orphaned-context GC are original to
azctx. Unlike the original project, azctx does not support Windows: the
isolation model is built on Unix process semantics (`$SHELL`, signals),
so only Linux and macOS binaries are published. azctx is not affiliated
with or endorsed by the original author.

## Contributing

This is an opinionated tool; issues and PRs are welcome, but features that
reintroduce mutation of `~/.azure` as a default won't be accepted.

## License

This project is licensed under the MIT License - see the LICENSE file for
details, which carries both the original author's copyright and this
project's.
