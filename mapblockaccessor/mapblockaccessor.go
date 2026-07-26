package mapblockaccessor

import (
	"fmt"
	"mapserver/db"
	"mapserver/eventbus"
	"mapserver/types"
	"sync"

	"time"

	cache "github.com/patrickmn/go-cache"
)

type MapBlockAccessor struct {
	accessor   db.DBAccessor
	blockcache *cache.Cache
	Eventbus   *eventbus.Eventbus
	maxcount   int
	loadLocks  [256]sync.Mutex
}

func (a *MapBlockAccessor) loadLock(pos *types.MapBlockCoords) *sync.Mutex {
	x := uint64(int64(pos.X))
	y := uint64(int64(pos.Y))
	z := uint64(int64(pos.Z))
	h := x*0x9e3779b185ebca87 ^ y*0xc2b2ae3d27d4eb4f ^ z*0x165667b19e3779f9
	return &a.loadLocks[byte(h^(h>>32))]
}

func getKey(pos *types.MapBlockCoords) string {
	return fmt.Sprintf("Coord %d/%d/%d", pos.X, pos.Y, pos.Z)
}

func NewMapBlockAccessor(accessor db.DBAccessor, expiretime, purgetime time.Duration, maxcount int) *MapBlockAccessor {
	blockcache := cache.New(expiretime, purgetime)

	return &MapBlockAccessor{
		accessor:   accessor,
		blockcache: blockcache,
		Eventbus:   eventbus.New(),
		maxcount:   maxcount,
	}
}
