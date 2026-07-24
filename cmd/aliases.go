package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/iul1an/azctx/pkg/subscription"
	"github.com/iul1an/azctx/pkg/types"
	"github.com/spf13/viper"
	yaml "go.yaml.in/yaml/v3"
)

// configuredAliases reads the optional `aliases:` map (alias to
// subscription ID or name) from the config file.
func configuredAliases() subscription.Aliases {
	return subscription.NewAliases(viper.GetStringMapString("aliases"))
}

// aliasIndex maps subscription ID to its sorted aliases, for the display
// side (list, status, completion).
func aliasIndex(cfg *types.Configuration) map[uuid.UUID][]string {
	aliases := configuredAliases()
	if len(aliases) == 0 || cfg == nil {
		return nil
	}
	sm := subscription.Manager{BaseManager: types.BaseManager{Configuration: cfg}, Aliases: aliases}
	return sm.AliasIndex()
}

// validateAliasKeys rejects alias keys that differ only in case: viper
// lowercases them into one, with an arbitrary winner.
func validateAliasKeys(data []byte, path string) error {
	var doc struct {
		Aliases map[string]any `yaml:"aliases"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	byLower := make(map[string][]string, len(doc.Aliases))
	for k := range doc.Aliases {
		lower := strings.ToLower(strings.TrimSpace(k))
		byLower[lower] = append(byLower[lower], k)
	}
	var collisions []string
	for lower, keys := range byLower {
		if len(keys) > 1 {
			sort.Strings(keys)
			quoted := make([]string, 0, len(keys))
			for _, k := range keys {
				quoted = append(quoted, fmt.Sprintf("%q", k))
			}
			collisions = append(collisions,
				fmt.Sprintf("%s all become %q", strings.Join(quoted, ", "), lower))
		}
	}
	if len(collisions) == 0 {
		return nil
	}
	sort.Strings(collisions)
	return fmt.Errorf("%s: alias keys differ only in case (%s); keep one of each",
		path, strings.Join(collisions, "; "))
}
