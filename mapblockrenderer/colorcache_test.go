package mapblockrenderer

import (
	"image/color"
	"sync"
	"testing"

	"github.com/minetest-go/colormapping"
)

type countingColorResolver struct {
	mutex  sync.Mutex
	calls  map[colorCacheKey]int
	colors map[colorCacheKey]*color.RGBA
}

func newCountingColorResolver() *countingColorResolver {
	return &countingColorResolver{
		calls:  make(map[colorCacheKey]int),
		colors: make(map[colorCacheKey]*color.RGBA),
	}
}

func (r *countingColorResolver) GetColor(name string, param2 int) *color.RGBA {
	key := colorCacheKey{name: name, param2: param2}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.calls[key]++
	return r.colors[key]
}

func (r *countingColorResolver) callCount(name string, param2 int) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.calls[colorCacheKey{name: name, param2: param2}]
}

func TestCachedColorResolverCachesHitsAndMisses(t *testing.T) {
	source := newCountingColorResolver()
	source.colors[colorCacheKey{name: "known", param2: 0}] = &color.RGBA{R: 1, G: 2, B: 3, A: 4}
	cache := newCachedColorResolver(source)

	for range 3 {
		got, found := cache.getColor("known", 0)
		if !found || got != (color.RGBA{R: 1, G: 2, B: 3, A: 4}) {
			t.Fatalf("unexpected known color: %#v, found=%v", got, found)
		}
		if _, found := cache.getColor("missing", 0); found {
			t.Fatal("missing color reported as found")
		}
	}

	if got := source.callCount("known", 0); got != 1 {
		t.Fatalf("known color resolved %d times, want 1", got)
	}
	if got := source.callCount("missing", 0); got != 1 {
		t.Fatalf("missing color resolved %d times, want 1", got)
	}
	if got := cache.count.Load(); got != 2 {
		t.Fatalf("cache contains %d entries, want one per distinct key", got)
	}
}

func TestCachedColorResolverIncludesParam2InKey(t *testing.T) {
	source := newCountingColorResolver()
	source.colors[colorCacheKey{name: "palette", param2: 1}] = &color.RGBA{R: 1}
	source.colors[colorCacheKey{name: "palette", param2: 2}] = &color.RGBA{R: 2}
	cache := newCachedColorResolver(source)

	one, _ := cache.getColor("palette", 1)
	two, _ := cache.getColor("palette", 2)
	if one.R != 1 || two.R != 2 {
		t.Fatalf("palette colors collapsed: %#v, %#v", one, two)
	}
}

func TestCachedColorResolverDoesNotExposeSharedColor(t *testing.T) {
	source := newCountingColorResolver()
	shared := &color.RGBA{R: 1, A: 17}
	source.colors[colorCacheKey{name: "known", param2: 0}] = shared
	cache := newCachedColorResolver(source)

	got, _ := cache.getColor("known", 0)
	got.A = 255
	again, _ := cache.getColor("known", 0)
	if shared.A != 17 || again.A != 17 {
		t.Fatalf("shared color mutated: source=%#v cached=%#v", shared, again)
	}
}

func TestCachedColorResolverResolvesConcurrentLookupOnce(t *testing.T) {
	source := newCountingColorResolver()
	source.colors[colorCacheKey{name: "known", param2: 7}] = &color.RGBA{R: 7}
	cache := newCachedColorResolver(source)

	var workers sync.WaitGroup
	for range 100 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			got, found := cache.getColor("known", 7)
			if !found || got.R != 7 {
				t.Errorf("unexpected concurrent result: %#v, found=%v", got, found)
			}
		}()
	}
	workers.Wait()

	if got := source.callCount("known", 7); got != 1 {
		t.Fatalf("concurrent color resolved %d times, want 1", got)
	}
}

func TestCachedColorResolverPreservesColorMappingResults(t *testing.T) {
	expectedSource := colormapping.NewColorMapping()
	if err := expectedSource.LoadDefaults(); err != nil {
		t.Fatal(err)
	}
	cachedSource := colormapping.NewColorMapping()
	if err := cachedSource.LoadDefaults(); err != nil {
		t.Fatal(err)
	}
	cache := newCachedColorResolver(cachedSource)

	tests := []colorCacheKey{
		{name: "scifi_nodes:blacktile2", param2: 0},
		{name: "mymod:my_red_node", param2: 0},
		{name: "unifiedbricks:brickblock_multicolor_dark", param2: 100},
		{name: "definitely:unmapped", param2: 0},
	}
	for _, key := range tests {
		want := expectedSource.GetColor(key.name, key.param2)
		got, found := cache.getColor(key.name, key.param2)
		if (want != nil) != found {
			t.Fatalf("%+v: found=%v, source=%#v", key, found, want)
		}
		if want != nil && got != *want {
			t.Fatalf("%+v: got %#v, want %#v", key, got, *want)
		}
	}
}

func BenchmarkColorResolution(b *testing.B) {
	mixedKeys := []colorCacheKey{
		{name: "scifi_nodes:blacktile2", param2: 0},
		{name: "unifiedbricks:brickblock_multicolor_dark", param2: 100},
		{name: "mymod:my_red_node", param2: 0},
		{name: "mineclonia:another_unmapped_node", param2: 0},
	}
	unknownKeys := []colorCacheKey{
		{name: "mineclonia:unknown_plain_node", param2: 0},
		{name: "mineclonia:another_unmapped_node", param2: 0},
		{name: "mineclonia:unmapped_stone_variant", param2: 0},
	}

	benchmarkDirectColors(b, "DirectMixed", mixedKeys)
	benchmarkCachedColors(b, "CachedMixed", mixedKeys)
	benchmarkDirectColors(b, "DirectUnknownHeavy", unknownKeys)
	benchmarkCachedColors(b, "CachedUnknownHeavy", unknownKeys)
	benchmarkParallelColors(b, "DirectUnknownParallel", unknownKeys, false)
	benchmarkParallelColors(b, "CachedUnknownParallel", unknownKeys, true)
}

func benchmarkParallelColors(b *testing.B, name string, keys []colorCacheKey, cached bool) {
	b.Run(name, func(b *testing.B) {
		source := colormapping.NewColorMapping()
		if err := source.LoadDefaults(); err != nil {
			b.Fatal(err)
		}
		var cache *cachedColorResolver
		if cached {
			cache = newCachedColorResolver(source)
			for _, key := range keys {
				cache.getColor(key.name, key.param2)
			}
		}
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := keys[i%len(keys)]
				if cached {
					cache.getColor(key.name, key.param2)
				} else {
					_ = source.GetColor(key.name, key.param2)
				}
				i++
			}
		})
	})
}

func benchmarkDirectColors(b *testing.B, name string, keys []colorCacheKey) {
	b.Run(name, func(b *testing.B) {
		source := colormapping.NewColorMapping()
		if err := source.LoadDefaults(); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			_ = source.GetColor(key.name, key.param2)
		}
	})
}

func benchmarkCachedColors(b *testing.B, name string, keys []colorCacheKey) {
	b.Run(name, func(b *testing.B) {
		source := colormapping.NewColorMapping()
		if err := source.LoadDefaults(); err != nil {
			b.Fatal(err)
		}
		cache := newCachedColorResolver(source)
		for _, key := range keys {
			cache.getColor(key.name, key.param2)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			cache.getColor(key.name, key.param2)
		}
	})
}
