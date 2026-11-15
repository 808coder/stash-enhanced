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

	// nil clears the setting, like an empty string
	tagID := ""
	if input.PreviouslyDeletedTagID != nil {
		tagID = strings.TrimSpace(*input.PreviouslyDeletedTagID)
	}
	c.SetString(config.TombstonesPreviouslyDeletedTagID, tagID)

	if err := c.Write(); err != nil {
		return makeConfigTombstonesResult(), err
	}

	return makeConfigTombstonesResult(), nil
}
