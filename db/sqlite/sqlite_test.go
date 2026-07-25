package sqlite

import (
	"database/sql"
	"encoding/json"
	"mapserver/coords"
	"mapserver/db"
	mapobjectsqlite "mapserver/mapobjectdb/sqlite"
	"mapserver/settings"
	"mapserver/testutils"
	"mapserver/types"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func collectIncrementalBlocks(t *testing.T, a *Sqlite3Accessor, limit int) []*db.Block {
	t.Helper()
	cursor := db.NewIncrementalCursor(99, nil)
	var blocks []*db.Block
	for {
		batch, err := a.FindBlocksByMtime(cursor, 101, limit)
		if err != nil {
			t.Fatal(err)
		}
		if len(batch) == 0 {
			break
		}
		blocks = append(blocks, batch...)
		last := batch[len(batch)-1]
		cursor = db.NewIncrementalCursor(last.Mtime, last.Pos)
		// Exercise the persisted-cursor restart path between every batch.
		encoded, err := json.Marshal(cursor)
		if err != nil {
			t.Fatal(err)
		}
		cursor = &db.IncrementalCursor{}
		if err := json.Unmarshal(encoded, cursor); err != nil {
			t.Fatal(err)
		}
	}
	return blocks
}

func TestFindBlocksByMtimePagesTimestampTiesXYZ(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestTimestampTiesXYZ.*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	rwdb, err := sql.Open("sqlite", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rwdb.Exec(`create table blocks (x int, y int, z int, data blob, mtime int)`); err != nil {
		t.Fatal(err)
	}
	for x := -2; x < 0; x++ {
		if _, err := rwdb.Exec(`insert into blocks(x,y,z,data,mtime) values(?,0,0,x'',100)`, x); err != nil {
			t.Fatal(err)
		}
	}
	for x := 0; x < 5; x++ {
		if _, err := rwdb.Exec(`insert into blocks(x,y,z,data,mtime) values(?,0,0,x'',101)`, x); err != nil {
			t.Fatal(err)
		}
	}
	rwdb.Close()

	a, err := New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Migrate(); err != nil {
		t.Fatal(err)
	}
	remainder, err := a.FindBlocksByMtime(
		db.NewIncrementalCursor(101, types.NewMapBlockCoords(1, 0, 0)),
		101,
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(remainder) != 2 || remainder[0].Pos.X != 2 || remainder[1].Pos.X != 3 {
		t.Fatalf("unexpected mid-group restart result: %#v", remainder)
	}

	blocks := collectIncrementalBlocks(t, a, 2)
	if len(blocks) != 7 {
		t.Fatalf("got %d blocks, want 7", len(blocks))
	}
	for i, block := range blocks {
		if block.Pos.X != i-2 || (i < 2 && block.Mtime != 100) || (i >= 2 && block.Mtime != 101) {
			t.Fatalf("block %d has x=%d", i, block.Pos.X)
		}
	}
}

func TestFindBlocksByMtimePagesTimestampTiesLegacy(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestTimestampTiesLegacy.*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	rwdb, err := sql.Open("sqlite", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rwdb.Exec(`create table blocks (pos int primary key, data blob, mtime int)`); err != nil {
		t.Fatal(err)
	}
	for x := -2; x < 0; x++ {
		pos := coords.CoordToPlain(types.NewMapBlockCoords(x, 0, 0))
		if _, err := rwdb.Exec(`insert into blocks(pos,data,mtime) values(?,x'',100)`, pos); err != nil {
			t.Fatal(err)
		}
	}
	for x := 0; x < 5; x++ {
		pos := coords.CoordToPlain(types.NewMapBlockCoords(x, 0, 0))
		if _, err := rwdb.Exec(`insert into blocks(pos,data,mtime) values(?,x'',101)`, pos); err != nil {
			t.Fatal(err)
		}
	}
	rwdb.Close()

	a, err := New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Migrate(); err != nil {
		t.Fatal(err)
	}

	remainder, err := a.FindBlocksByMtime(
		db.NewIncrementalCursor(101, types.NewMapBlockCoords(1, 0, 0)),
		101,
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(remainder) != 2 || remainder[0].Pos.X != 2 || remainder[1].Pos.X != 3 {
		t.Fatalf("unexpected legacy mid-group restart result: %#v", remainder)
	}

	blocks := collectIncrementalBlocks(t, a, 2)
	if len(blocks) != 7 {
		t.Fatalf("got %d blocks, want 7", len(blocks))
	}
	seen := make(map[int]bool)
	for _, block := range blocks {
		if seen[block.Pos.X] {
			t.Fatalf("duplicate x=%d", block.Pos.X)
		}
		seen[block.Pos.X] = true
	}
}

func TestSQLiteWatermarkProtectsLateVisibility(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestLateVisibility.*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	rwdb, err := sql.Open("sqlite", tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer rwdb.Close()
	if _, err := rwdb.Exec(`create table blocks (x int, y int, z int, data blob, mtime int)`); err != nil {
		t.Fatal(err)
	}
	a, err := New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer a.db.Close()
	if err := a.Migrate(); err != nil {
		t.Fatal(err)
	}

	tx, err := rwdb.Begin()
	if err != nil {
		t.Fatal(err)
	}
	txMtime := time.Now().Unix()
	if _, err := tx.Exec(`insert into blocks(x,y,z,data,mtime) values(1,2,3,x'',?)`, txMtime); err != nil {
		t.Fatal(err)
	}
	watermark, err := a.GetIncrementalWatermark(2 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if watermark.UpperMtime >= txMtime {
		t.Fatalf("watermark %d passed active timestamp %d", watermark.UpperMtime, txMtime)
	}
	blocks, err := a.FindBlocksByMtime(db.NewIncrementalCursor(0, nil), txMtime, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 0 {
		t.Fatal("uncommitted SQLite block became visible")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	blocks, err = a.FindBlocksByMtime(db.NewIncrementalCursor(watermark.UpperMtime, nil), txMtime, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || *blocks[0].Pos != *types.NewMapBlockCoords(1, 2, 3) {
		t.Fatalf("late-visible block was not returned: %#v", blocks)
	}
}

func TestLegacyInitialRenderRecountsWithoutNegativeProgress(t *testing.T) {
	worldfile, err := os.CreateTemp("", "TestLegacyInitialProgress.*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(worldfile.Name())
	worldfile.Close()
	if err := testutils.CreateTestDatabaseLegacy(worldfile.Name()); err != nil {
		t.Fatal(err)
	}

	a, err := New(worldfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Migrate(); err != nil {
		t.Fatal(err)
	}

	statefile, err := os.CreateTemp("", "TestLegacyInitialProgressState.*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(statefile.Name())
	statefile.Close()
	stateDB, err := mapobjectsqlite.New(statefile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if err := stateDB.Migrate(); err != nil {
		t.Fatal(err)
	}
	s := settings.New(stateDB)
	layers := []*types.Layer{{Id: 0, From: types.MinCoord, To: types.MaxCoord}}
	if err := settings.ResetInitialRender(s, layers); err != nil {
		t.Fatal(err)
	}

	result, err := a.FindNextInitialBlocks(s, layers, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Progress < 0 {
		t.Fatalf("progress=%f, want non-negative", result.Progress)
	}
	if got := s.GetInt64(settings.SETTING_TOTAL_LEGACY_COUNT, -1); got <= 0 {
		t.Fatalf("legacy total=%d, want positive recount", got)
	}
}

func TestMigrateEmpty(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestMigrateEmpty.*.sqlite")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	testutils.CreateEmptyDatabase(tmpfile.Name())
	a, err := New(tmpfile.Name())
	if err != nil {
		panic(err)
	}
	err = a.Migrate()
	if err != nil {
		panic(err)
	}
}

func TestMigrateEmptyLegacy(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestMigrateEmpty.*.sqlite")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	testutils.CreateEmptyLegacyDatabase(tmpfile.Name())
	a, err := New(tmpfile.Name())
	if err != nil {
		panic(err)
	}
	err = a.Migrate()
	if err != nil {
		panic(err)
	}
}

func TestMigrate(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestMigrate.*.sqlite")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	testutils.CreateEmptyDatabase(tmpfile.Name())
	a, err := New(tmpfile.Name())
	if err != nil {
		panic(err)
	}
	err = a.Migrate()
	if err != nil {
		panic(err)
	}
}

func TestMigrateAndQuery(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestMigrateAndQuery.*.sqlite")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	testutils.CreateTestDatabase(tmpfile.Name())
	a, err := New(tmpfile.Name())
	if err != nil {
		panic(err)
	}

	err = a.Migrate()
	if err != nil {
		panic(err)
	}

	block, err := a.GetBlock(types.NewMapBlockCoords(0, 0, 0))

	if err != nil {
		panic(err)
	}

	if block == nil {
		t.Fatal("no data")
	}

}

func TestMigrateAndQueryCount(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestMigrateAndQueryStride.*.sqlite")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	testutils.CreateTestDatabase(tmpfile.Name())
	a, err := New(tmpfile.Name())
	if err != nil {
		panic(err)
	}

	err = a.Migrate()
	if err != nil {
		panic(err)
	}

	count, err := a.CountBlocks()
	if err != nil {
		panic(err)
	}

	if count <= 0 {
		t.Fatal("zero count")
	}
}

func TestMigrateAndQueryCountLegacy(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestMigrateAndQueryStride.*.sqlite")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	testutils.CreateTestDatabaseLegacy(tmpfile.Name())
	a, err := New(tmpfile.Name())
	if err != nil {
		panic(err)
	}

	err = a.Migrate()
	if err != nil {
		panic(err)
	}

	count, err := a.CountBlocks()
	if err != nil {
		panic(err)
	}

	if count <= 0 {
		t.Fatal("zero count")
	}
}

func TestMigrateAndQueryTimestamp(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "TestMigrateAndQueryStride.*.sqlite")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	testutils.CreateTestDatabase(tmpfile.Name())
	a, err := New(tmpfile.Name())
	if err != nil {
		panic(err)
	}

	err = a.Migrate()
	if err != nil {
		panic(err)
	}

	count, err := a.GetTimestamp()
	if err != nil {
		panic(err)
	}

	if count <= 0 {
		t.Fatal("zero count")
	}
}
