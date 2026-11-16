package config

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// Scene tombstone options. Kept in their own file to limit merge conflicts
// with upstream config.go.
const (
	// TombstonesRecord enables recording tombstones for deleted scenes.
	// Defaults to true.
	TombstonesRecord = "tombstones.record"
	// TombstonesPreviouslyDeletedTagID is the id of the tag added to a
	// rescanned scene that was manually deleted before. Empty means a tag
	// named "Previously deleted" is found or created on demand.
	TombstonesPreviouslyDeletedTagID = "tombstones.previously_deleted_tag_id"
	// TombstonesReasons is the list of selectable deletion reasons.
	TombstonesReasons = "tombstones.reasons"
)

// TombstoneReasonColor is the badge colour of a deletion reason. The values
// name colours from a fixed palette; the UI maps them to theme variants.
type TombstoneReasonColor string

const (
	TombstoneReasonColorNeutral TombstoneReasonColor = "NEUTRAL"
	TombstoneReasonColorBlue    TombstoneReasonColor = "BLUE"
	TombstoneReasonColorTeal    TombstoneReasonColor = "TEAL"
	TombstoneReasonColorGreen   TombstoneReasonColor = "GREEN"
	TombstoneReasonColorYellow  TombstoneReasonColor = "YELLOW"
	TombstoneReasonColorRed     TombstoneReasonColor = "RED"
)

var AllTombstoneReasonColor = []TombstoneReasonColor{
	TombstoneReasonColorNeutral,
	TombstoneReasonColorBlue,
	TombstoneReasonColorTeal,
	TombstoneReasonColorGreen,
	TombstoneReasonColorYellow,
	TombstoneReasonColorRed,
}

func (e TombstoneReasonColor) IsValid() bool {
	return slices.Contains(AllTombstoneReasonColor, e)
}

func (e TombstoneReasonColor) String() string {
	return string(e)
}

func (e *TombstoneReasonColor) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = TombstoneReasonColor(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid TombstoneReasonColor", str)
	}
	return nil
}

func (e TombstoneReasonColor) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// TombstoneReason is a selectable deletion reason with its badge colour.
type TombstoneReason struct {
	Name  string               `json:"name" yaml:"name" koanf:"name"`
	Color TombstoneReasonColor `json:"color" yaml:"color" koanf:"color"`
}

// defaultTombstonesReasons are used when no reasons are configured.
var defaultTombstonesReasons = []TombstoneReason{
	{Name: "Not my taste", Color: TombstoneReasonColorNeutral},
	{Name: "Low quality", Color: TombstoneReasonColorYellow},
	{Name: "Replaced by a better version", Color: TombstoneReasonColorGreen},
	{Name: "Duplicate", Color: TombstoneReasonColorTeal},
	{Name: "Broken file", Color: TombstoneReasonColorYellow},
}

// GetTombstonesReasons returns the selectable deletion reasons, in order.
// Falls back to the default list when unset or unreadable, for example when
// the config file still holds the former plain string list. An unknown
// colour read from the file is shown as neutral.
func (i *Config) GetTombstonesReasons() []*TombstoneReason {
	i.RLock()
	defer i.RUnlock()

	v := i.forKey(TombstonesReasons)
	if !v.Exists(TombstonesReasons) {
		return defaultTombstoneReasons()
	}

	var reasons []*TombstoneReason
	if err := v.Unmarshal(TombstonesReasons, &reasons); err != nil {
		// not logged: this is read on every configuration query
		return defaultTombstoneReasons()
	}

	for _, r := range reasons {
		if r != nil && !r.Color.IsValid() {
			r.Color = TombstoneReasonColorNeutral
		}
	}

	ret, err := NormaliseTombstoneReasons(reasons)
	if err != nil {
		return defaultTombstoneReasons()
	}
	return ret
}

func defaultTombstoneReasons() []*TombstoneReason {
	ret := make([]*TombstoneReason, len(defaultTombstonesReasons))
	for j, r := range defaultTombstonesReasons {
		ret[j] = &r
	}
	return ret
}

// ErrNoTombstoneReasons is returned when a reasons list is empty after
// normalising: at least one deletion reason is required.
var ErrNoTombstoneReasons = errors.New("at least one deletion reason is required")

// NormaliseTombstoneReasons trims each name, drops entries with an empty name
// and drops case-insensitive duplicates, keeping the first. It returns
// ErrNoTombstoneReasons if nothing is left, and an error for an invalid
// colour. The input is not modified.
func NormaliseTombstoneReasons(in []*TombstoneReason) ([]*TombstoneReason, error) {
	seen := make(map[string]struct{}, len(in))
	ret := make([]*TombstoneReason, 0, len(in))
	for _, p := range in {
		if p == nil {
			continue
		}
		r := *p
		r.Name = strings.TrimSpace(r.Name)
		if r.Name == "" {
			continue
		}
		if !r.Color.IsValid() {
			return nil, fmt.Errorf("deletion reason %q: %q is not a valid colour", r.Name, r.Color)
		}
		k := strings.ToLower(r.Name)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		ret = append(ret, &r)
	}
	if len(ret) == 0 {
		return nil, ErrNoTombstoneReasons
	}
	return ret, nil
}

// GetTombstonesRecord returns true if tombstones are recorded when scenes are
// deleted. Defaults to true.
func (i *Config) GetTombstonesRecord() bool {
	return i.getBoolDefault(TombstonesRecord, true)
}

// GetTombstonesPreviouslyDeletedTagID returns the configured id of the
// "previously deleted" tag, or an empty string if none is configured.
func (i *Config) GetTombstonesPreviouslyDeletedTagID() string {
	return i.getString(TombstonesPreviouslyDeletedTagID)
}
