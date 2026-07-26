package mapblockrenderer

import (
	"image/color"
	"sync"
	"sync/atomic"
)

type colorResolver interface {
	GetColor(name string, param2 int) *color.RGBA
}

type colorCacheKey struct {
	name   string
	param2 int
}

type colorCacheEntry struct {
	once  sync.Once
	color color.RGBA
	found bool
}

type cachedColorResolver struct {
	resolver colorResolver
	entries  sync.Map
	count    atomic.Int64
}

func newCachedColorResolver(resolver colorResolver) *cachedColorResolver {
	return &cachedColorResolver{
		resolver: resolver,
	}
}

func (r *cachedColorResolver) getColor(name string, param2 int) (color.RGBA, bool) {
	key := colorCacheKey{name: name, param2: param2}
	if entryValue, ok := r.entries.Load(key); ok {
		entry := entryValue.(*colorCacheEntry)
		entry.once.Do(func() {
			r.resolve(entry, name, param2)
		})
		return entry.color, entry.found
	}

	entryValue, loaded := r.entries.LoadOrStore(key, &colorCacheEntry{})
	if !loaded {
		r.count.Add(1)
	}
	entry := entryValue.(*colorCacheEntry)

	entry.once.Do(func() {
		r.resolve(entry, name, param2)
	})
	return entry.color, entry.found
}

func (r *cachedColorResolver) resolve(entry *colorCacheEntry, name string, param2 int) {
	resolved := r.resolver.GetColor(name, param2)
	if resolved != nil {
		entry.color = *resolved
		entry.found = true
	}
}
