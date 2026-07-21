// Package state persists azctx's context-switch history. It lives in its
// own file under $XDG_STATE_HOME (not in ~/.azctx.yml) so that recording
// state never rewrites the user's config — writing through the main viper
// instance would serialize every bound flag along with it.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// StateManager handles all state operations.
type StateManager interface {
	// GetCurrentContext returns the most recently picked subscription, or
	// empty strings if nothing was ever picked.
	GetCurrentContext() (id string, name string)
	// GetLastContext returns the previously used subscription (the one
	// before the most recent pick), or empty strings if there is none.
	GetLastContext() (id string, name string)
	// RecordSwitch records a successful switch to the given subscription,
	// rotating the previous current into the last slot (cd - semantics).
	RecordSwitch(id string, name string) error
}

type contextRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type stateFile struct {
	Current contextRef `json:"current"`
	Last    contextRef `json:"last"`
}

// FileStateManager stores state as JSON under
// ${XDG_STATE_HOME:-~/.local/state}/azctx/state.json.
type FileStateManager struct {
	path string
}

func NewFileStateManager() *FileStateManager {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return &FileStateManager{path: filepath.Join(dir, "azctx", "state.json")}
}

func (f *FileStateManager) read() stateFile {
	var s stateFile
	data, err := os.ReadFile(f.path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

func (f *FileStateManager) GetCurrentContext() (string, string) {
	s := f.read()
	return s.Current.ID, s.Current.Name
}

func (f *FileStateManager) GetLastContext() (string, string) {
	s := f.read()
	return s.Last.ID, s.Last.Name
}

func (f *FileStateManager) RecordSwitch(id string, name string) error {
	s := f.read()
	if s.Current.ID == id {
		// Re-picking the current subscription is not a switch; keep the
		// previous slot intact.
		return nil
	}
	if s.Current.ID != "" {
		s.Last = s.Current
	}
	s.Current = contextRef{ID: id, Name: name}

	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, data, 0o600)
}
