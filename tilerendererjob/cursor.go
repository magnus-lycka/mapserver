package tilerendererjob

import (
	"encoding/json"
	"fmt"
	"mapserver/db"
	"mapserver/settings"

	"github.com/sirupsen/logrus"
)

func loadIncrementalCursor(s settings.Settings) (*db.IncrementalCursor, error) {
	fallback := func(err error) (*db.IncrementalCursor, error) {
		logrus.WithError(err).Warn("Ignoring invalid incremental cursor; falling back to last_mtime")
		return &db.IncrementalCursor{
			Version: db.IncrementalCursorVersion,
			Mtime:   s.GetInt64(settings.SETTING_LAST_MTIME, 0),
		}, nil
	}
	value := s.GetString(settings.SETTING_INCREMENTAL_CURSOR, "")
	if value == "" {
		return &db.IncrementalCursor{
			Version: db.IncrementalCursorVersion,
			Mtime:   s.GetInt64(settings.SETTING_LAST_MTIME, 0),
		}, nil
	}

	var cursor db.IncrementalCursor
	if err := json.Unmarshal([]byte(value), &cursor); err != nil {
		return fallback(fmt.Errorf("decode incremental cursor: %w", err))
	}
	if cursor.Version != db.IncrementalCursorVersion {
		return fallback(fmt.Errorf("unsupported incremental cursor version: %d", cursor.Version))
	}
	if cursor.PositionValid && cursor.Pos == nil {
		return fallback(fmt.Errorf("incremental cursor has no position"))
	}
	return &cursor, nil
}

func saveIncrementalCursor(s settings.Settings, cursor *db.IncrementalCursor) error {
	value, err := json.Marshal(cursor)
	if err != nil {
		return fmt.Errorf("encode incremental cursor: %w", err)
	}

	// The complete compound cursor is the authoritative, atomic setting.
	s.SetString(settings.SETTING_INCREMENTAL_CURSOR, string(value))
	// Keep the legacy scalar updated for downgrade compatibility.
	s.SetInt64(settings.SETTING_LAST_MTIME, cursor.Mtime)
	return nil
}
