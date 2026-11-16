package api

import (
	"context"
	"strings"

	"github.com/stashapp/stash/internal/manager/config"
)

func makeConfigTombstonesResult() *ConfigTombstonesResult {
	c := config.GetInstance()

	ret := &ConfigTombstonesResult{
		Record:  c.GetTombstonesRecord(),
		Reasons: c.GetTombstonesReasons(),
	}
	if id := c.GetTombstonesPreviouslyDeletedTagID(); id != "" {
		ret.PreviouslyDeletedTagID = &id
	}
	return ret
}

func (r *mutationResolver) ConfigureTombstones(ctx context.Context, input ConfigTombstonesInput) (*ConfigTombstonesResult, error) {
	c := config.GetInstance()

	// validate before changing anything, so a rejected request leaves the
	// configuration untouched
	var reasons []*config.TombstoneReason
	if input.Reasons != nil {
		var err error
		if reasons, err = config.NormaliseTombstoneReasons(input.Reasons); err != nil {
			return makeConfigTombstonesResult(), err
		}
	}

	r.setConfigBool(config.TombstonesRecord, input.Record)

	if reasons != nil {
		c.SetInterface(config.TombstonesReasons, reasons)
	}

	// nil leaves the setting unchanged; an empty string clears it
	if input.PreviouslyDeletedTagID != nil {
		c.SetString(config.TombstonesPreviouslyDeletedTagID, strings.TrimSpace(*input.PreviouslyDeletedTagID))
	}

	if err := c.Write(); err != nil {
		return makeConfigTombstonesResult(), err
	}

	return makeConfigTombstonesResult(), nil
}
