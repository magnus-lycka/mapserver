package mapblockaccessor

import (
	"io/ioutil"
	"mapserver/db"
	"mapserver/db/sqlite"
	"mapserver/testutils"
	"mapserver/types"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestSimpleAccess(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	tmpfile, err := ioutil.TempFile("", "TestMigrate.*.sqlite")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())
	testutils.CreateTestDatabase(tmpfile.Name())

	a, err := sqlite.New(tmpfile.Name())
	if err != nil {
		panic(err)
	}

	err = a.Migrate()
	if err != nil {
		panic(err)
	}

	cache := NewMapBlockAccessor(a, 500*time.Millisecond, 1000*time.Millisecond, 1000)
	mb, err := cache.GetMapBlock(types.NewMapBlockCoords(0, 0, 0))

	if err != nil {
		panic(err)
	}

	if mb == nil {
		t.Fatal("Mapblock is nil")
	}
}

func TestFindMapBlocksAdvancesRawCursorWhenLayerFiltersEverything(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestFilteredIncremental.*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()
	if err := testutils.CreateTestDatabase(tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	a, err := sqlite.New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Migrate(); err != nil {
		t.Fatal(err)
	}
	cache := NewMapBlockAccessor(a, time.Second, time.Second, 10)
	watermark, err := a.GetIncrementalWatermark(0)
	if err != nil {
		t.Fatal(err)
	}
	result, err := cache.FindMapBlocksByMtime(
		db.NewIncrementalCursor(-1, nil),
		watermark.UpperMtime,
		10,
		[]*types.Layer{{Id: 0, From: 100, To: 101}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.UnfilteredCount == 0 {
		t.Fatal("expected raw database rows")
	}
	if len(result.List) != 0 {
		t.Fatalf("got %d rendered rows, want 0", len(result.List))
	}
	if result.LastCursor == nil || !result.LastCursor.PositionValid {
		t.Fatal("raw-row cursor did not advance")
	}
}
