package types

import "encoding/json"

// azureProfile.json is owned by the Azure CLI and contains fields azctx knows
// nothing about (environmentName, tenantDefaultDomain, ...). To avoid
// corrupting the profile on rewrite (upstream issue #36), Configuration,
// Subscription and Tenant capture the raw JSON they were unmarshaled from
// and overlay their typed fields onto it when marshaled back, so unknown
// fields survive the round trip.

// mergeRaw overlays the typed object's keys onto the captured raw keys.
func mergeRaw(raw map[string]json.RawMessage, typed []byte) ([]byte, error) {
	if raw == nil {
		return typed, nil
	}
	var typedMap map[string]json.RawMessage
	if err := json.Unmarshal(typed, &typedMap); err != nil {
		return nil, err
	}
	merged := make(map[string]json.RawMessage, len(raw)+len(typedMap))
	for k, v := range raw {
		merged[k] = v
	}
	for k, v := range typedMap {
		merged[k] = v
	}
	return json.Marshal(merged)
}

func (c *Configuration) UnmarshalJSON(data []byte) error {
	type alias Configuration
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*c = Configuration(a)
	return json.Unmarshal(data, &c.raw)
}

func (c Configuration) MarshalJSON() ([]byte, error) {
	type alias Configuration
	typed, err := json.Marshal(alias(c))
	if err != nil {
		return nil, err
	}
	return mergeRaw(c.raw, typed)
}

func (s *Subscription) UnmarshalJSON(data []byte) error {
	type alias Subscription
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = Subscription(a)
	return json.Unmarshal(data, &s.raw)
}

func (s Subscription) MarshalJSON() ([]byte, error) {
	type alias Subscription
	typed, err := json.Marshal(alias(s))
	if err != nil {
		return nil, err
	}
	return mergeRaw(s.raw, typed)
}

func (t *Tenant) UnmarshalJSON(data []byte) error {
	type alias Tenant
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*t = Tenant(a)
	return json.Unmarshal(data, &t.raw)
}

func (t Tenant) MarshalJSON() ([]byte, error) {
	type alias Tenant
	typed, err := json.Marshal(alias(t))
	if err != nil {
		return nil, err
	}
	return mergeRaw(t.raw, typed)
}
