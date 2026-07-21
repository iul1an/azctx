package isolation

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// metaFileName is the marker file written into every tempdir by Setup. It
// both identifies the dir as azctx-owned and records who owns it.
const metaFileName = ".azctx-meta.json"

type meta struct {
	PID     int       `json:"pid"`
	Started time.Time `json:"started"`
}

// Context describes one isolated azctx config dir found under $TMPDIR.
type Context struct {
	Dir          string    `json:"configDir"`
	PID          int       `json:"pid"`
	Started      time.Time `json:"started"`
	Alive        bool      `json:"alive"`
	Active       bool      `json:"active"`
	Subscription string    `json:"subscription,omitempty"`
}

// ListContexts returns every azctx-owned tempdir, live or orphaned. Dirs
// matching the name pattern but lacking the meta marker are not ours and are
// ignored.
func ListContexts() ([]Context, error) {
	dirs, err := filepath.Glob(filepath.Join(os.TempDir(), "azctx.*"))
	if err != nil {
		return nil, err
	}
	active := os.Getenv("AZURE_CONFIG_DIR")
	var out []Context
	for _, dir := range dirs {
		st, err := os.Stat(dir)
		if err != nil || !st.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, metaFileName))
		if err != nil {
			continue
		}
		var m meta
		if json.Unmarshal(data, &m) != nil || m.PID <= 0 {
			continue
		}
		out = append(out, Context{
			Dir:          dir,
			PID:          m.PID,
			Started:      m.Started,
			Alive:        processAlive(m.PID),
			Active:       dir == active,
			Subscription: defaultSubscription(dir),
		})
	}
	return out, nil
}

// Sweep removes orphaned contexts (owning process dead) and returns how many
// were removed. The active context is never touched, and neither is anything
// without an azctx meta marker. Best-effort by design.
func Sweep() int {
	ctxs, err := ListContexts()
	if err != nil {
		return 0
	}
	n := 0
	for _, c := range ctxs {
		if !c.Alive && !c.Active {
			if os.RemoveAll(c.Dir) == nil {
				n++
			}
		}
	}
	return n
}

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// defaultSubscription reads the default subscription's name from the dir's
// azureProfile.json; "" if none is set or the file is unreadable.
func defaultSubscription(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "azureProfile.json"))
	if err != nil {
		return ""
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var p struct {
		Subscriptions []struct {
			Name      string `json:"name"`
			IsDefault bool   `json:"isDefault"`
		} `json:"subscriptions"`
	}
	if json.Unmarshal(data, &p) != nil {
		return ""
	}
	for _, s := range p.Subscriptions {
		if s.IsDefault {
			return s.Name
		}
	}
	return ""
}
