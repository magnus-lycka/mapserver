package mapblockaccessor

import (
	"mapserver/db"
	"mapserver/settings"
	"mapserver/types"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type blockingAccessor struct {
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
}

func (a *blockingAccessor) Migrate() error               { return nil }
func (a *blockingAccessor) GetTimestamp() (int64, error) { return 0, nil }
func (a *blockingAccessor) GetIncrementalWatermark(time.Duration) (*db.IncrementalWatermark, error) {
	return &db.IncrementalWatermark{}, nil
}
func (a *blockingAccessor) FindBlocksByMtime(*db.IncrementalCursor, int64, int) ([]*db.Block, error) {
	return nil, nil
}
func (a *blockingAccessor) FindNextInitialBlocks(settings.Settings, []*types.Layer, int) (*db.InitialBlocksResult, error) {
	return &db.InitialBlocksResult{}, nil
}
func (a *blockingAccessor) GetBlock(*types.MapBlockCoords) (*db.Block, error) {
	a.calls.Add(1)
	a.started <- struct{}{}
	<-a.release
	return nil, nil
}

func waitForLoad(t *testing.T, started <-chan struct{}) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for block load")
	}
}

func TestDifferentMapBlocksLoadConcurrently(t *testing.T) {
	backend := &blockingAccessor{started: make(chan struct{}, 2), release: make(chan struct{})}
	accessor := NewMapBlockAccessor(backend, time.Minute, time.Minute, 100)
	first := types.NewMapBlockCoords(0, 0, 0)
	second := types.NewMapBlockCoords(1, 0, 0)
	if accessor.loadLock(first) == accessor.loadLock(second) {
		t.Fatal("test coordinates unexpectedly share a lock stripe")
	}

	var workers sync.WaitGroup
	workers.Add(2)
	for _, pos := range []*types.MapBlockCoords{first, second} {
		go func() {
			defer workers.Done()
			if _, err := accessor.GetMapBlock(pos); err != nil {
				t.Errorf("GetMapBlock: %v", err)
			}
		}()
	}
	waitForLoad(t, backend.started)
	waitForLoad(t, backend.started)
	close(backend.release)
	workers.Wait()
}

func TestSameMapBlockLoadsOnce(t *testing.T) {
	backend := &blockingAccessor{started: make(chan struct{}, 2), release: make(chan struct{})}
	accessor := NewMapBlockAccessor(backend, time.Minute, time.Minute, 100)
	pos := types.NewMapBlockCoords(2, 3, 4)

	var workers sync.WaitGroup
	workers.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer workers.Done()
			if _, err := accessor.GetMapBlock(pos); err != nil {
				t.Errorf("GetMapBlock: %v", err)
			}
		}()
	}
	waitForLoad(t, backend.started)
	close(backend.release)
	workers.Wait()
	if got := backend.calls.Load(); got != 1 {
		t.Fatalf("backend calls = %d, want 1", got)
	}
}
