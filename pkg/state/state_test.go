package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T) *FileStateManager {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	return NewFileStateManager()
}

func TestRecordSwitchRotation(t *testing.T) {
	sm := newTestManager(t)

	// Nothing recorded yet: no previous context.
	id, name := sm.GetLastContext()
	assert.Empty(t, id)
	assert.Empty(t, name)

	// First pick: becomes current, still no previous.
	require.NoError(t, sm.RecordSwitch("id-a", "sub-a"))
	id, _ = sm.GetLastContext()
	assert.Empty(t, id)

	// Second pick: the first rotates into previous.
	require.NoError(t, sm.RecordSwitch("id-b", "sub-b"))
	id, name = sm.GetLastContext()
	assert.Equal(t, "id-a", id)
	assert.Equal(t, "sub-a", name)

	// Switching back (cd - style): slots swap.
	require.NoError(t, sm.RecordSwitch("id-a", "sub-a"))
	id, name = sm.GetLastContext()
	assert.Equal(t, "id-b", id)
	assert.Equal(t, "sub-b", name)
}

func TestRecordSwitchSameContextKeepsPrevious(t *testing.T) {
	sm := newTestManager(t)
	require.NoError(t, sm.RecordSwitch("id-a", "sub-a"))
	require.NoError(t, sm.RecordSwitch("id-b", "sub-b"))

	// Re-picking the current subscription must not clobber the previous slot.
	require.NoError(t, sm.RecordSwitch("id-b", "sub-b"))
	id, _ := sm.GetLastContext()
	assert.Equal(t, "id-a", id)
}

func TestStatePersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	sm := NewFileStateManager()
	require.NoError(t, sm.RecordSwitch("id-a", "sub-a"))
	require.NoError(t, sm.RecordSwitch("id-b", "sub-b"))

	// Fresh instance (new process simulation) reads the same state.
	sm2 := NewFileStateManager()
	id, name := sm2.GetLastContext()
	assert.Equal(t, "id-a", id)
	assert.Equal(t, "sub-a", name)

	// State file is private and lives under XDG_STATE_HOME.
	info, err := os.Stat(filepath.Join(dir, "azctx", "state.json"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
