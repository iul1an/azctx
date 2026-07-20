// Package isolation implements per-shell Azure config isolation. It copies
// ~/.azure into a private tempdir, points AZURE_CONFIG_DIR at the copy, and
// spawns a subshell scoped to it, so the master ~/.azure is never mutated.
package isolation

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

const tempDirPattern = "aztx.*"

// IsActive reports whether the current process is already running inside an
// aztx isolated context, i.e. AZURE_CONFIG_DIR points at an aztx tempdir.
func IsActive() bool {
	dir := os.Getenv("AZURE_CONFIG_DIR")
	if dir == "" {
		return false
	}
	prefix := filepath.Join(os.TempDir(), "aztx.")
	return strings.HasPrefix(dir, prefix)
}

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

	tmpDir, err := os.MkdirTemp("", tempDirPattern)
	if err != nil {
		return "", fmt.Errorf("creating isolated config dir: %w", err)
	}

	if err := os.CopyFS(tmpDir, os.DirFS(azureDir)); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("copying %s to isolated config dir: %w", azureDir, err)
	}

	if err := os.Setenv("AZURE_CONFIG_DIR", tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("setting AZURE_CONFIG_DIR: %w", err)
	}
	return tmpDir, nil
}

// SpawnShell runs $SHELL (fallback /bin/zsh) attached to the current
// terminal, inheriting the environment (including AZURE_CONFIG_DIR set by
// Setup). It blocks until the shell exits so the caller's deferred cleanup
// can remove the tempdir. SIGINT/SIGTERM are swallowed while the shell runs;
// they are delivered to the shell's foreground process by the terminal.
func SpawnShell() error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}

	cmd := exec.Command(shell)
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
			// The shell exiting non-zero (e.g. last command failed) is not an
			// aztx error; don't surface it as one.
			return nil
		}
		return fmt.Errorf("running %s: %w", shell, err)
	}
	return nil
}
