package postgres

import (
	"context"
	"encoding/json"
	"mapserver/db"
	"mapserver/types"
	"os"
	"testing"
	"time"
)

func newIntegrationAccessor(t *testing.T) *PostgresAccessor {
	t.Helper()
	dsn := os.Getenv("MAPSERVER_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("MAPSERVER_TEST_POSTGRES_DSN is not set")
	}

	a, err := New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.db.Exec(`drop table if exists blocks`); err != nil {
		t.Fatal(err)
	}
	if _, err := a.db.Exec(`
		create table blocks (
			posx integer not null,
			posy integer not null,
			posz integer not null,
			data bytea,
			mtime bigint not null,
			primary key (posx,posy,posz)
		);
		create index blocks_time on blocks(mtime);
	`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		a.db.Exec(`drop table if exists blocks`)
		a.db.Close()
	})
	return a
}

func TestPostgresPagesTimestampTies(t *testing.T) {
	a := newIntegrationAccessor(t)
	for x := -2; x < 0; x++ {
		if _, err := a.db.Exec(`insert into blocks(posx,posy,posz,data,mtime) values($1,0,0,'',100)`, x); err != nil {
			t.Fatal(err)
		}
	}
	for x := 0; x < 5; x++ {
		if _, err := a.db.Exec(`insert into blocks(posx,posy,posz,data,mtime) values($1,0,0,'',101)`, x); err != nil {
			t.Fatal(err)
		}
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

	cursor := db.NewIncrementalCursor(99, nil)
	var got []*db.Block
	for {
		batch, err := a.FindBlocksByMtime(cursor, 101, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(batch) == 0 {
			break
		}
		got = append(got, batch...)
		last := batch[len(batch)-1]
		cursor = db.NewIncrementalCursor(last.Mtime, last.Pos)
		encoded, err := json.Marshal(cursor)
		if err != nil {
			t.Fatal(err)
		}
		cursor = &db.IncrementalCursor{}
		if err := json.Unmarshal(encoded, cursor); err != nil {
			t.Fatal(err)
		}
	}

	if len(got) != 7 {
		t.Fatalf("got %d blocks, want 7", len(got))
	}
	for i, block := range got {
		if block.Pos.X != i-2 {
			t.Fatalf("block %d has x=%d", i, block.Pos.X)
		}
	}
}

func TestPostgresWatermarkAlwaysAppliesSafetyLag(t *testing.T) {
	a := newIntegrationAccessor(t)
	watermark, err := a.GetIncrementalWatermark(2 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var afterMtime int64
	if err := a.db.QueryRow(`select floor(extract(epoch from now()) * 1000)::bigint`).Scan(&afterMtime); err != nil {
		t.Fatal(err)
	}
	if watermark.UpperMtime > afterMtime-2000 {
		t.Fatalf("watermark %d did not apply safety lag below %d", watermark.UpperMtime, afterMtime)
	}
}

func TestPostgresWatermarkProtectsLateCommit(t *testing.T) {
	a := newIntegrationAccessor(t)
	ctx := context.Background()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var txMtime int64
	if err := tx.QueryRowContext(ctx, `select floor(extract(epoch from now()) * 1000)::bigint`).Scan(&txMtime); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `insert into blocks(posx,posy,posz,data,mtime) values(1,2,3,'',$1)`, txMtime); err != nil {
		t.Fatal(err)
	}

	watermark, err := a.GetIncrementalWatermark(0)
	if err != nil {
		t.Fatal(err)
	}
	if watermark.UpperMtime >= txMtime {
		t.Fatalf("watermark %d passed open transaction %d in mode %q", watermark.UpperMtime, txMtime, watermark.Mode)
	}
	blocks, err := a.FindBlocksByMtime(db.NewIncrementalCursor(0, nil), watermark.UpperMtime, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 0 {
		t.Fatal("uncommitted block became visible")
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	watermark, err = a.GetIncrementalWatermark(0)
	if err != nil {
		t.Fatal(err)
	}
	blocks, err = a.FindBlocksByMtime(db.NewIncrementalCursor(0, nil), watermark.UpperMtime, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || *blocks[0].Pos != *types.NewMapBlockCoords(1, 2, 3) {
		t.Fatalf("late commit was not returned: %#v", blocks)
	}
}
