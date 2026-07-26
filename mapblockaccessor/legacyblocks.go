package mapblockaccessor

import (
	"mapserver/eventbus"
	"mapserver/settings"
	"mapserver/types"

	"github.com/minetest-go/mapparser"
	cache "github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

type FindNextLegacyBlocksResult struct {
	HasMore         bool
	List            []*types.ParsedMapblock
	UnfilteredCount int
	Progress        float64
	LastMtime       int64
}

func (a *MapBlockAccessor) prepareCacheForBatch(incomingItems int) {
	if a.maxcount <= 0 {
		return
	}
	cachedItems := a.blockcache.ItemCount()
	if cachedItems == 0 || cachedItems+incomingItems <= a.maxcount {
		return
	}
	fields := logrus.Fields{
		"cached items":   cachedItems,
		"incoming items": incomingItems,
		"maxcount":       a.maxcount,
	}
	logrus.WithFields(fields).Debug("Flushing cache before batch")
	a.blockcache.Flush()
}

func (a *MapBlockAccessor) FindNextLegacyBlocks(s settings.Settings, layers []*types.Layer, limit int) (*FindNextLegacyBlocksResult, error) {

	nextResult, err := a.accessor.FindNextInitialBlocks(s, layers, limit)

	if err != nil {
		return nil, err
	}

	blocks := nextResult.List
	result := FindNextLegacyBlocksResult{}

	// Keep the incoming spatial batch resident until it has been rendered.
	a.prepareCacheForBatch(len(blocks))

	mblist := make([]*types.ParsedMapblock, 0)
	result.HasMore = nextResult.HasMore
	result.UnfilteredCount = nextResult.UnfilteredCount
	result.Progress = nextResult.Progress
	result.LastMtime = nextResult.LastMtime

	for _, block := range blocks {

		fields := logrus.Fields{
			"x": block.Pos.X,
			"y": block.Pos.Y,
			"z": block.Pos.Z,
		}
		logrus.WithFields(fields).Trace("mapblock")

		key := getKey(block.Pos)

		mapblock, err := mapparser.Parse(block.Data)
		if err != nil {
			fields := logrus.Fields{
				"x":   block.Pos.X,
				"y":   block.Pos.Y,
				"z":   block.Pos.Z,
				"err": err,
			}
			logrus.WithFields(fields).Error("mapblock-parse")

			return nil, err
		}

		a.Eventbus.Emit(eventbus.MAPBLOCK_RENDERED, types.NewParsedMapblock(mapblock, block.Pos))

		if a.maxcount > 0 {
			a.blockcache.Set(key, mapblock, cache.DefaultExpiration)
			cacheBlockCount.Inc()
		}
		mblist = append(mblist, types.NewParsedMapblock(mapblock, block.Pos))

	}

	result.List = mblist

	fields := logrus.Fields{
		"len(List)":       len(result.List),
		"unfilteredCount": result.UnfilteredCount,
		"hasMore":         result.HasMore,
		"limit":           limit,
	}
	logrus.WithFields(fields).Debug("FindMapBlocksByPos:Result")

	return &result, nil
}
