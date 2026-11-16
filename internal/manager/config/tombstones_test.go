package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_TombstonesDefaults(t *testing.T) {
	i := InitializeEmpty()

	assert.True(t, i.GetTombstonesRecord())
	assert.Empty(t, i.GetTombstonesPreviouslyDeletedTagID())
}

func TestConfig_TombstonesReasonsDefaults(t *testing.T) {
	i := InitializeEmpty()

	assert.Equal(t, []string{
		"Not my taste",
		"Low quality",
		"Replaced by a better version",
		"Duplicate",
		"Broken file",
	}, i.GetTombstonesReasons())
}

func TestConfig_TombstonesReasonsSet(t *testing.T) {
	i := InitializeEmpty()

	i.SetInterface(TombstonesReasons, []string{"A", "B"})

	assert.Equal(t, []string{"A", "B"}, i.GetTombstonesReasons())
}

func TestNormaliseTombstoneReasons(t *testing.T) {
	got, err := NormaliseTombstoneReasons([]string{"  Duplicate ", "", "   ", "duplicate", "Low quality", "LOW QUALITY"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"Duplicate", "Low quality"}, got)
}

func TestNormaliseTombstoneReasons_rejectsEmptyList(t *testing.T) {
	for _, in := range [][]string{{}, {" ", ""}, {"\t"}} {
		got, err := NormaliseTombstoneReasons(in)
		assert.ErrorIs(t, err, ErrNoTombstoneReasons, "input %q", in)
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
