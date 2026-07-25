package settings

import (
	"fmt"
	"mapserver/coords"
	"mapserver/types"
)

const (
	SETTING_LAST_MTIME             = "last_mtime"
	SETTING_INCREMENTAL_CURSOR     = "incremental_cursor_v1"
	SETTING_INITIAL_RUN            = "initial_run"
	SETTING_LEGACY_PROCESSED       = "legacy_processed"
	SETTING_LAST_LAYER             = "last_layer"
	SETTING_LAST_X_BLOCK           = "last_x_block"
	SETTING_LAST_Y_BLOCK           = "last_y_block"
	SETTING_LAST_POS               = "last_pos"
	SETTING_TOTAL_LEGACY_COUNT     = "total_legacy_count"
	SETTING_PROCESSED_LEGACY_COUNT = "total_processed_legacy_count"
)

type Settings interface {
	GetString(key string, defaultValue string) string
	SetString(key string, value string)
	GetInt(key string, defaultValue int) int
	SetInt(key string, value int)
	GetInt64(key string, defaultValue int64) int64
	SetInt64(key string, value int64)
	GetBool(key string, defaultValue bool) bool
	SetBool(key string, value bool)
}

func ResetInitialRender(s Settings, layers []*types.Layer) error {
	if len(layers) == 0 {
		return fmt.Errorf("cannot reset initial rendering without layers")
	}

	lowestLayer := layers[0].Id
	for _, layer := range layers[1:] {
		if layer.Id < lowestLayer {
			lowestLayer = layer.Id
		}
	}

	s.SetBool(SETTING_INITIAL_RUN, true)
	s.SetInt(SETTING_LAST_LAYER, lowestLayer)
	s.SetInt(SETTING_LAST_X_BLOCK, -129)
	s.SetInt(SETTING_LAST_Y_BLOCK, -128)
	s.SetInt64(SETTING_LAST_POS, coords.MinPlainCoord-1)
	s.SetInt64(SETTING_TOTAL_LEGACY_COUNT, -1)
	s.SetInt64(SETTING_PROCESSED_LEGACY_COUNT, 0)
	return nil
}
