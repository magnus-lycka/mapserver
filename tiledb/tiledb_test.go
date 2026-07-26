package tiledb

import (
	"fmt"
	"mapserver/coords"
	"os"
	"testing"
)

func TestTileDB(t *testing.T) {
	tmpfile, err := os.MkdirTemp("", "TestTileDB")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpfile)

	db, err := New(tmpfile)
	if err != nil {
		panic(err)
	}

	c := coords.NewTileCoords(0, 0, 1, 2)

	err = db.SetTile(c, []byte{1, 2, 3})
	if err != nil {
		panic(err)
	}

	tile, err := db.GetTile(c)
	if err != nil {
		panic(err)
	}

	if len(tile) != 3 {
		t.Error("wrong size")
	}

	c2 := coords.NewTileCoords(1, 0, 1, 2)
	tile, err = db.GetTile(c2)
	if err != nil {
		panic(err)
	}

	if tile != nil {
		t.Error("tile exists")
	}

}

func TestListTiles(t *testing.T) {
	dir := t.TempDir()
	db, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []*coords.TileCoords{
		coords.NewTileCoords(-3, 4, 9, 2),
		coords.NewTileCoords(7, -8, 9, 2),
	}
	for _, tc := range want {
		if err := db.SetTile(tc, []byte("png")); err != nil {
			t.Fatal(err)
		}
	}
	// A different zoom must not be included.
	if err := db.SetTile(coords.NewTileCoords(1, 1, 8, 2), []byte("png")); err != nil {
		t.Fatal(err)
	}

	got, err := db.ListTiles(2, 9)
	if err != nil {
		t.Fatal(err)
	}
	gotKeys := make(map[string]bool, len(got))
	for _, tc := range got {
		gotKeys[fmt.Sprintf("%d/%d/%d/%d", tc.X, tc.Y, tc.Zoom, tc.LayerId)] = true
	}
	for _, tc := range want {
		key := fmt.Sprintf("%d/%d/%d/%d", tc.X, tc.Y, tc.Zoom, tc.LayerId)
		if !gotKeys[key] {
			t.Errorf("missing tile %s", key)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("got %d tiles, want %d", len(got), len(want))
	}

	empty, err := db.ListTiles(99, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("got %d tiles for missing layer", len(empty))
	}
}
