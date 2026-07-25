package tilerendererjob

import (
	"mapserver/db"
	"mapserver/mapobjectdb/sqlite"
	"mapserver/settings"
	"mapserver/types"
	"os"
	"testing"
)

func newTestSettings(t *testing.T) settings.Settings {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "IncrementalCursor.*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	t.Cleanup(func() { os.Remove(tmpfile.Name()) })

	objectdb, err := sqlite.New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if err := objectdb.Migrate(); err != nil {
		t.Fatal(err)
	}
	return settings.New(objectdb)
}

func TestLoadIncrementalCursorFallsBackToLastMtime(t *testing.T) {
	s := newTestSettings(t)
	s.SetInt64(settings.SETTING_LAST_MTIME, 123)

	cursor, err := loadIncrementalCursor(s)
	if err != nil {
		t.Fatal(err)
	}
	if cursor.Mtime != 123 || cursor.PositionValid {
		t.Fatalf("unexpected cursor: %#v", cursor)
	}
}

func TestIncrementalCursorRoundTrip(t *testing.T) {
	s := newTestSettings(t)
	want := db.NewIncrementalCursor(456, types.NewMapBlockCoords(-3, 4, 5))
	if err := saveIncrementalCursor(s, want); err != nil {
		t.Fatal(err)
	}

	got, err := loadIncrementalCursor(s)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mtime != want.Mtime || !got.PositionValid || *got.Pos != *want.Pos {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	if s.GetInt64(settings.SETTING_LAST_MTIME, 0) != want.Mtime {
		t.Fatal("legacy last_mtime was not updated")
	}
}

func TestLoadIncrementalCursorFallsBackFromInvalidStoredCursor(t *testing.T) {
	for _, value := range []string{
		`not-json`,
		`{"version":2,"mtime":456}`,
		`{"version":1,"mtime":456,"position_valid":true}`,
	} {
		t.Run(value, func(t *testing.T) {
			s := newTestSettings(t)
			s.SetInt64(settings.SETTING_LAST_MTIME, 123)
			s.SetString(settings.SETTING_INCREMENTAL_CURSOR, value)

			cursor, err := loadIncrementalCursor(s)
			if err != nil {
				t.Fatal(err)
			}
			if cursor.Mtime != 123 || cursor.PositionValid {
				t.Fatalf("unexpected fallback cursor: %#v", cursor)
			}
		})
	}
}
