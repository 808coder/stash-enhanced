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

func TestConfig_TombstonesSettings(t *testing.T) {
	i := InitializeEmpty()

	i.SetBool(TombstonesRecord, false)
	i.SetString(TombstonesPreviouslyDeletedTagID, "42")

	assert.False(t, i.GetTombstonesRecord())
	assert.Equal(t, "42", i.GetTombstonesPreviouslyDeletedTagID())
}
