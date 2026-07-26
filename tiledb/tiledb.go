package tiledb

import (
	"fmt"
	"mapserver/coords"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func New(path string) (*TileDB, error) {
	return &TileDB{
		path: path,
	}, nil
}

// ListTiles returns persisted tiles at one layer and zoom. It is used to build
// the lower initial-render pyramid once the independently rendered zoom-9
// regions are complete, and also makes that final phase restart-safe.
func (tdb *TileDB) ListTiles(layerID, zoom int) ([]*coords.TileCoords, error) {
	root := fmt.Sprintf("%s/%d/%d", tdb.path, layerID, zoom)
	tiles := make([]*coords.TileCoords, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".png" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) != 2 {
			return nil
		}
		x, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil
		}
		y, err := strconv.Atoi(strings.TrimSuffix(parts[1], ".png"))
		if err != nil {
			return nil
		}
		tiles = append(tiles, coords.NewTileCoords(x, y, zoom, layerID))
		return nil
	})
	if os.IsNotExist(err) {
		return tiles, nil
	}
	return tiles, err
}

type TileDB struct {
	path string
}

func (tdb *TileDB) getDirAndFile(pos *coords.TileCoords) (string, string) {
	dir := fmt.Sprintf("%s/%d/%d/%d", tdb.path, pos.LayerId, pos.Zoom, pos.X)
	file := fmt.Sprintf("%s/%d.png", dir, pos.Y)
	return dir, file
}

func (tdb *TileDB) GetTile(pos *coords.TileCoords) ([]byte, error) {
	_, file := tdb.getDirAndFile(pos)
	info, _ := os.Stat(file)
	if info != nil {
		content, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}

		return content, err
	}

	return nil, nil
}

func (tdb *TileDB) SetTile(pos *coords.TileCoords, tile []byte) error {
	dir, file := tdb.getDirAndFile(pos)
	os.MkdirAll(dir, 0700)

	err := os.WriteFile(file, tile, 0644)
	return err
}
