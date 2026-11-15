package api

import (
	"context"
	"strings"

	"github.com/stashapp/stash/internal/manager/config"
)

func makeConfigTombstonesResult() *ConfigTombstonesResult {
	c := config.GetInstance()

	ret := &ConfigTombstonesResult{
		Record: c.GetTombstonesRecord(),
	}
	if id := c.GetTombstonesPreviouslyDeletedTagID(); id != "" {
		ret.PreviouslyDeletedTagID = &id
	}
	return ret
}

func (r *mutationResolver) ConfigureTombstones(ctx context.Context, input ConfigTombstonesInput) (*ConfigTombstonesResult, error) {
	c := config.GetInstance()

	r.setConfigBool(config.TombstonesRecord, input.Record)

	// nil leaves the setting unchanged; an empty string clears it
	if input.PreviouslyDeletedTagID != nil {
		c.SetString(config.TombstonesPreviouslyDeletedTagID, strings.TrimSpace(*input.PreviouslyDeletedTagID))
	}

	if err := c.Write(); err != nil {
		return makeConfigTombstonesResult(), err
	}

	return makeConfigTombstonesResult(), nil
}
