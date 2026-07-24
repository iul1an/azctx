package subscription

import (
	"sort"
	"strings"

	"github.com/google/uuid"
)

// Aliases maps a user-defined alias to a subscription ID or name, as
// configured under `aliases:` in the config file.
type Aliases map[string]string

// NewAliases lowercases and trims keys, dropping blank ones, blank targets,
// and keys holding a tab or newline (they break `azctx list` columns).
func NewAliases(raw map[string]string) Aliases {
	if len(raw) == 0 {
		return nil
	}
	a := make(Aliases, len(raw))
	for k, v := range raw {
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		if k == "" || v == "" || strings.ContainsAny(k, "\t\n\r") {
			continue
		}
		a[k] = v
	}
	if len(a) == 0 {
		return nil
	}
	return a
}

// lookup returns the target an alias points at, if query is an alias.
func (a Aliases) lookup(query string) (string, bool) {
	if len(a) == 0 {
		return "", false
	}
	target, ok := a[strings.ToLower(strings.TrimSpace(query))]
	return target, ok
}

// AliasIndex maps subscription ID to its sorted aliases. Aliases whose
// target matches no subscription are left out; selecting one is an error.
func (sm *Manager) AliasIndex() map[uuid.UUID][]string {
	idx := make(map[uuid.UUID][]string, len(sm.Aliases))
	for alias, target := range sm.Aliases {
		sub, err := sm.lookup(target)
		if err != nil {
			continue
		}
		idx[sub.ID] = append(idx[sub.ID], alias)
	}
	for id := range idx {
		sort.Strings(idx[id])
	}
	return idx
}

// AliasesFor returns the sorted aliases of a single subscription.
func (sm *Manager) AliasesFor(id uuid.UUID) []string {
	return sm.AliasIndex()[id]
}
