// Package isolation implements per-shell Azure config isolation. It copies
// ~/.azure into a private tempdir, points AZURE_CONFIG_DIR at the copy, and
// spawns a subshell scoped to it, so the master ~/.azure is never mutated.
package isolation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const tempDirPattern = "azctx.*"

// IsActive reports whether the current process is already running inside an
// azctx isolated context, i.e. AZURE_CONFIG_DIR points at an azctx tempdir.
func IsActive() bool {
	dir := os.Getenv("AZURE_CONFIG_DIR")
	if dir == "" {
		return false
	}
	prefix := filepath.Join(os.TempDir(), "azctx.")
	return strings.HasPrefix(dir, prefix)
}

// newContextDir creates a private tempdir with the azctx meta marker written
// before anything else, so Sweep/ListContexts never see an unmarked
// half-built dir of ours.
func newContextDir() (string, error) {
	tmpDir, err := os.MkdirTemp("", tempDirPattern)
	if err != nil {
		return "", fmt.Errorf("creating isolated config dir: %w", err)
	}

	metaData, err := json.Marshal(meta{PID: os.Getpid(), Started: time.Now().Truncate(time.Second)})
	if err == nil {
		err = os.WriteFile(filepath.Join(tmpDir, metaFileName), metaData, 0o600)
	}
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("writing context metadata: %w", err)
	}
	return tmpDir, nil
}

func activate(tmpDir string) (string, error) {
	if err := os.Setenv("AZURE_CONFIG_DIR", tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("setting AZURE_CONFIG_DIR: %w", err)
	}
	// az's telemetry uploader outlives the command and recreates the context
	// dir to log into it, after azctx has removed it.
	if err := os.Setenv(telemetryEnv, "0"); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("setting %s: %w", telemetryEnv, err)
	}
	return tmpDir, nil
}

// telemetryEnv is az's [core] collect_telemetry config key in env form
// (knack maps AZURE_<SECTION>_<OPTION>).
const telemetryEnv = "AZURE_CORE_COLLECT_TELEMETRY"

// Setup copies ~/.azure into a fresh private tempdir and sets
// AZURE_CONFIG_DIR to it for this process (and any children it spawns).
// It returns the tempdir path; the caller is responsible for removing it.
func Setup() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	azureDir := filepath.Join(home, ".azure")
	if _, err := os.Stat(azureDir); err != nil {
		return "", fmt.Errorf("azure config directory %s not found (run `az login` first): %w", azureDir, err)
	}

	tmpDir, err := newContextDir()
	if err != nil {
		return "", err
	}
	if err := copyDir(azureDir, tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("copying %s to isolated config dir: %w", azureDir, err)
	}
	return activate(tmpDir)
}

// skipDirs are az's write-only diagnostic dirs; logs/ is unbounded.
var skipDirs = map[string]bool{"logs": true, "commands": true}

// copyDir copies src into dst, carrying file modes across and resolving
// symlinks into regular files, neither of which os.CopyFS does.
func copyDir(src, dst string) error {
	return copyTree(src, dst, map[string]bool{})
}

// copyTree walks src, recursing into symlinked dirs itself (WalkDir does
// not); seen holds resolved paths so a symlink loop cannot spin forever.
func copyTree(src, dst string, seen map[string]bool) error {
	resolved, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	if seen[resolved] {
		return nil
	}
	seen[resolved] = true

	return filepath.WalkDir(resolved, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(resolved, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil // dst exists and stays 0700 whatever src is
		}
		if d.IsDir() && skipDirs[rel] {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)

		info, err := os.Stat(path) // follows symlinks
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return err
			}
			if err := os.Chmod(target, info.Mode().Perm()); err != nil { // MkdirAll honors umask
				return err
			}
			if d.Type()&fs.ModeSymlink != 0 {
				return copyTree(path, target, seen)
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil // sockets and devices carry no state worth copying
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Chmod(mode); err != nil { // OpenFile honors umask
		_ = out.Close()
		return err
	}
	return out.Close()
}

// SetupEmpty creates a fresh, empty isolated config dir — nothing is copied
// from ~/.azure. az behaves as never-logged-in inside it, for ephemeral
// workflows where the login should vanish with the context.
func SetupEmpty() (string, error) {
	tmpDir, err := newContextDir()
	if err != nil {
		return "", err
	}
	return activate(tmpDir)
}

// SpawnShell runs $SHELL (fallback /bin/zsh) attached to the current
// terminal, inheriting the environment (including AZURE_CONFIG_DIR set by
// Setup). It blocks until the shell exits so the caller's deferred cleanup
// can remove the tempdir. The shell's own exit status is not treated as an
// azctx error.
func SpawnShell() error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	_, err := RunCommand([]string{shell})
	return err
}

// RunCommand runs argv attached to the current terminal, inheriting the
// environment (including AZURE_CONFIG_DIR set by Setup), and blocks until it
// exits so the caller's deferred cleanup can remove the tempdir. It returns
// the command's exit code; err is only set when the command could not be
// started. SIGINT/SIGTERM are swallowed while the child runs; the terminal
// delivers them to the child's foreground process group.
func RunCommand(argv []string) (int, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)
	go func() {
		for range sigs {
		}
	}()

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return -1, fmt.Errorf("running %s: %w", argv[0], err)
	}
	return 0, nil
}
