package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_TombstonesDefaults(t *testing.T) {
	i := InitializeEmpty()

	assert.True(t, i.GetTombstonesRecord())
	assert.Empty(t, i.GetTombstonesPreviouslyDeletedTagID())
}

func reason(name string, color TombstoneReasonColor) *TombstoneReason {
	return &TombstoneReason{Name: name, Color: color}
}

var expectedDefaultReasons = []*TombstoneReason{
	reason("Not my taste", TombstoneReasonColorNeutral),
	reason("Low quality", TombstoneReasonColorYellow),
	reason("Replaced by a better version", TombstoneReasonColorGreen),
	reason("Duplicate", TombstoneReasonColorTeal),
	reason("Broken file", TombstoneReasonColorYellow),
}

// loadConfigYAML loads a config file the way startup does.
func loadConfigYAML(t *testing.T, content []byte) *Config {
	t.Helper()
	fn := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(fn, content, 0600); err != nil {
		t.Fatal(err)
	}
	i := InitializeEmpty()
	if err := i.load(fn); err != nil {
		t.Fatal(err)
	}
	return i
}

func TestConfig_TombstonesReasonsDefaults(t *testing.T) {
	i := InitializeEmpty()

	assert.Equal(t, expectedDefaultReasons, i.GetTombstonesReasons())
}

func TestConfig_TombstonesReasonsDefaultsAreCopies(t *testing.T) {
	i := InitializeEmpty()

	i.GetTombstonesReasons()[0].Name = "changed"

	assert.Equal(t, expectedDefaultReasons, i.GetTombstonesReasons())
}

func TestConfig_TombstonesReasonsSet(t *testing.T) {
	i := InitializeEmpty()
	want := []*TombstoneReason{
		reason("A", TombstoneReasonColorRed),
		reason("B", TombstoneReasonColorBlue),
	}

	i.SetInterface(TombstonesReasons, want)

	assert.Equal(t, want, i.GetTombstonesReasons())
}

func TestConfig_TombstonesReasonsRoundTripThroughFile(t *testing.T) {
	i := InitializeEmpty()
	want := []*TombstoneReason{
		reason("A", TombstoneReasonColorRed),
		reason("B", TombstoneReasonColorNeutral),
	}
	i.SetInterface(TombstonesReasons, want)

	data, err := i.Marshal()
	assert.NoError(t, err)
	assert.Contains(t, string(data), "name: A")
	assert.Contains(t, string(data), "color: RED")

	assert.Equal(t, want, loadConfigYAML(t, data).GetTombstonesReasons())
}

func TestConfig_TombstonesReasonsFromFile(t *testing.T) {
	i := loadConfigYAML(t, []byte(`tombstones:
  reasons:
    - name: " Duplicate "
      color: TEAL
    - name: Unknown colour
      color: PURPLE
    - name: No colour
    - name: ""
      color: RED
`))

	assert.Equal(t, []*TombstoneReason{
		reason("Duplicate", TombstoneReasonColorTeal),
		reason("Unknown colour", TombstoneReasonColorNeutral),
		reason("No colour", TombstoneReasonColorNeutral),
	}, i.GetTombstonesReasons())
}

// The former format was a plain list of names. It is not migrated; the
// defaults apply until the setting is saved again.
func TestConfig_TombstonesReasonsOldStringListFallsBackToDefaults(t *testing.T) {
	i := loadConfigYAML(t, []byte(`tombstones:
  reasons:
    - A
    - B
`))

	assert.Equal(t, expectedDefaultReasons, i.GetTombstonesReasons())
}

func TestConfig_TombstonesReasonsUnusableFallsBackToDefaults(t *testing.T) {
	for _, content := range []string{
		"tombstones:\n  reasons: []\n",
		"tombstones:\n  reasons: Duplicate\n",
		"tombstones:\n  reasons:\n    - name: \" \"\n",
	} {
		i := loadConfigYAML(t, []byte(content))
		assert.Equal(t, expectedDefaultReasons, i.GetTombstonesReasons(), "content %q", content)
	}
}

func TestNormaliseTombstoneReasons(t *testing.T) {
	in := []*TombstoneReason{
		reason("  Duplicate ", TombstoneReasonColorTeal),
		reason("", TombstoneReasonColorRed),
		reason("   ", TombstoneReasonColorRed),
		nil,
		reason("duplicate", TombstoneReasonColorRed),
		reason("Low quality", TombstoneReasonColorYellow),
		reason("LOW QUALITY", TombstoneReasonColorNeutral),
	}

	got, err := NormaliseTombstoneReasons(in)

	assert.NoError(t, err)
	assert.Equal(t, []*TombstoneReason{
		reason("Duplicate", TombstoneReasonColorTeal),
		reason("Low quality", TombstoneReasonColorYellow),
	}, got)
	assert.Equal(t, "  Duplicate ", in[0].Name, "input must not be modified")
}

func TestNormaliseTombstoneReasons_rejectsEmptyList(t *testing.T) {
	for _, in := range [][]*TombstoneReason{
		{},
		{reason(" ", TombstoneReasonColorNeutral), reason("", TombstoneReasonColorNeutral)},
		{reason("\t", TombstoneReasonColorNeutral)},
	} {
		got, err := NormaliseTombstoneReasons(in)
		assert.ErrorIs(t, err, ErrNoTombstoneReasons, "input %v", in)
		assert.Nil(t, got)
	}
}

func TestNormaliseTombstoneReasons_rejectsInvalidColor(t *testing.T) {
	for _, c := range []TombstoneReasonColor{"", "PURPLE", "red"} {
		got, err := NormaliseTombstoneReasons([]*TombstoneReason{reason("A", c)})
		assert.Error(t, err, "colour %q", c)
		assert.Nil(t, got)
	}
}

func TestConfig_TombstonesSettings(t *testing.T) {
	i := InitializeEmpty()

	i.SetBool(TombstonesRecord, false)
	i.SetString(TombstonesPreviouslyDeletedTagID, "42")

	assert.False(t, i.GetTombstonesRecord())
	assert.Equal(t, "42", i.GetTombstonesPreviouslyDeletedTagID())
}
