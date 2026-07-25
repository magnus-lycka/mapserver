package db

import (
	"mapserver/settings"
	"mapserver/types"
	"time"
)

type Block struct {
	Pos   *types.MapBlockCoords
	Data  []byte
	Mtime int64
}

type InitialBlocksResult struct {
	List            []*Block
	UnfilteredCount int
	HasMore         bool
	Progress        float64
	LastMtime       int64
}

const IncrementalCursorVersion = 1

type IncrementalCursor struct {
	Version       int                   `json:"version"`
	Mtime         int64                 `json:"mtime"`
	Pos           *types.MapBlockCoords `json:"pos,omitempty"`
	PositionValid bool                  `json:"position_valid"`
}

func NewIncrementalCursor(mtime int64, pos *types.MapBlockCoords) *IncrementalCursor {
	return &IncrementalCursor{
		Version:       IncrementalCursorVersion,
		Mtime:         mtime,
		Pos:           pos,
		PositionValid: pos != nil,
	}
}

type IncrementalWatermark struct {
	UpperMtime int64
	Mode       string
}

type DBAccessor interface {
	Migrate() error

	GetTimestamp() (int64, error)
	GetIncrementalWatermark(safetyLag time.Duration) (*IncrementalWatermark, error)
	FindBlocksByMtime(cursor *IncrementalCursor, upperMtime int64, limit int) ([]*Block, error)
	FindNextInitialBlocks(s settings.Settings, layers []*types.Layer, limit int) (*InitialBlocksResult, error)
	GetBlock(pos *types.MapBlockCoords) (*Block, error)
}
