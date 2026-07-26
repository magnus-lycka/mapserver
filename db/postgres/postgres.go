package postgres

import (
	"database/sql"
	"embed"
	"mapserver/db"
	"mapserver/types"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type PostgresAccessor struct {
	db                     *sql.DB
	watermarkWarningLock   sync.Mutex
	watermarkWarningActive bool
	lastWatermarkWarning   time.Time
}

const maxOpenConnections = 16

//go:embed migrations/*.sql
var migrations embed.FS

func (db *PostgresAccessor) Migrate() error {
	hasMtime := true
	_, err := db.db.Query("select max(mtime) from blocks")
	if err != nil {
		hasMtime = false
	}

	if !hasMtime {
		log.Info("Migrating database, this might take a while depending on the mapblock-count")
		start := time.Now()

		sql, err := migrations.ReadFile("migrations/postgres_mapdb_migrate.sql")
		if err != nil {
			return err
		}

		_, err = db.db.Exec(string(sql))
		if err != nil {
			return err
		}
		t := time.Now()
		elapsed := t.Sub(start)
		log.WithFields(logrus.Fields{"elapsed": elapsed}).Info("Migration completed")
	}

	return nil
}

func convertRows(posx, posy, posz int, data []byte, mtime int64) *db.Block {
	c := types.NewMapBlockCoords(posx, posy, posz)
	return &db.Block{Pos: c, Data: data, Mtime: mtime}
}

func (a *PostgresAccessor) FindBlocksByMtime(cursor *db.IncrementalCursor, upperMtime int64, limit int) ([]*db.Block, error) {
	blocks := make([]*db.Block, 0)
	if upperMtime < cursor.Mtime || limit <= 0 {
		return blocks, nil
	}
	var x, y, z int
	if cursor.PositionValid && cursor.Pos != nil {
		x, y, z = cursor.Pos.X, cursor.Pos.Y, cursor.Pos.Z
	}

	if cursor.PositionValid {
		rows, err := a.db.Query(
			getBlocksAtCursorMtimeQuery,
			cursor.Mtime,
			x, y, z,
			limit,
		)
		if err != nil {
			return nil, err
		}
		blocks, err = appendBlocks(rows, blocks)
		rows.Close()
		if err != nil || len(blocks) == limit {
			return blocks, err
		}
	}

	rows, err := a.db.Query(getBlocksByMtimeQuery, upperMtime, cursor.Mtime, limit-len(blocks))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return appendBlocks(rows, blocks)
}

func appendBlocks(rows *sql.Rows, blocks []*db.Block) ([]*db.Block, error) {
	for rows.Next() {
		var posx, posy, posz int
		var data []byte
		var mtime int64
		if err := rows.Scan(&posx, &posy, &posz, &data, &mtime); err != nil {
			return nil, err
		}
		blocks = append(blocks, convertRows(posx, posy, posz, data, mtime))
	}
	return blocks, rows.Err()
}

func (a *PostgresAccessor) GetIncrementalWatermark(safetyLag time.Duration) (*db.IncrementalWatermark, error) {
	row := a.db.QueryRow(getIncrementalWatermarkQuery)
	var nowMtime int64
	var oldestSameRole, oldestAll sql.NullInt64
	var hasReadAllStats bool
	if err := row.Scan(&nowMtime, &oldestSameRole, &oldestAll, &hasReadAllStats); err != nil {
		return nil, err
	}

	lagMillis := safetyLag.Milliseconds()
	if safetyLag > 0 && lagMillis == 0 {
		lagMillis = 1
	}

	watermark := &db.IncrementalWatermark{
		UpperMtime: nowMtime - lagMillis,
		Mode:       "time-lag",
	}

	oldest := oldestSameRole
	if hasReadAllStats {
		oldest = oldestAll
		watermark.Mode = "time-lag+all-visible-xact-start"
	} else if oldestSameRole.Valid {
		watermark.Mode = "time-lag+same-role-xact-start"
	}

	if oldest.Valid && oldest.Int64-1 < watermark.UpperMtime {
		watermark.UpperMtime = oldest.Int64 - 1
	}
	a.warnIfWatermarkStalled(nowMtime, lagMillis, watermark, oldest)

	return watermark, nil
}

func (a *PostgresAccessor) warnIfWatermarkStalled(nowMtime, lagMillis int64, watermark *db.IncrementalWatermark, oldest sql.NullInt64) {
	const warningThreshold = int64((30 * time.Second) / time.Millisecond)
	const warningInterval = 5 * time.Minute

	stalled := oldest.Valid && nowMtime-watermark.UpperMtime > lagMillis+warningThreshold
	a.watermarkWarningLock.Lock()
	defer a.watermarkWarningLock.Unlock()

	if !stalled {
		if a.watermarkWarningActive {
			log.Info("Incremental watermark is advancing normally again")
		}
		a.watermarkWarningActive = false
		return
	}

	now := time.Now()
	if !a.watermarkWarningActive || now.Sub(a.lastWatermarkWarning) >= warningInterval {
		log.WithFields(logrus.Fields{
			"holdback":    time.Duration(nowMtime-watermark.UpperMtime) * time.Millisecond,
			"oldestXact":  oldest.Int64,
			"safetyLag":   time.Duration(lagMillis) * time.Millisecond,
			"watermark":   watermark.UpperMtime,
			"warningMode": watermark.Mode,
		}).Warn("Long-running transaction is holding back incremental rendering")
		a.lastWatermarkWarning = now
	}
	a.watermarkWarningActive = true
}

func (a *PostgresAccessor) CountBlocks(frommtime, tomtime int64) (int, error) {
	rows, err := a.db.Query(countBlocksQuery, frommtime, tomtime)
	if err != nil {
		panic(err)
	}

	defer rows.Close()

	if rows.Next() {
		var count int64

		err = rows.Scan(&count)
		if err != nil {
			return 0, err
		}

		return int(count), nil
	}

	return 0, nil
}

func (a *PostgresAccessor) GetTimestamp() (int64, error) {
	rows, err := a.db.Query(getTimestampQuery)
	if err != nil {
		return 0, err
	}

	defer rows.Close()

	if rows.Next() {
		var ts float64

		err = rows.Scan(&ts)
		if err != nil {
			return 0, err
		}

		return int64(ts), nil
	}

	return 0, nil
}

func (a *PostgresAccessor) GetBlock(pos *types.MapBlockCoords) (*db.Block, error) {
	rows, err := a.db.Query(getBlockQuery, pos.X, pos.Y, pos.Z)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() {
		var posx, posy, posz int
		var data []byte
		var mtime int64

		err = rows.Scan(&posx, &posy, &posz, &data, &mtime)
		if err != nil {
			return nil, err
		}

		mb := convertRows(posx, posy, posz, data, mtime)
		return mb, nil
	}

	return nil, nil
}

func New(connStr string) (*PostgresAccessor, error) {
	db, err := sql.Open("postgres", connStr+" sslmode=disable")
	if err != nil {
		return nil, err
	}
	// Renderer workers issue concurrent point reads. Bound and retain the pool:
	// an unlimited pool can churn through local TCP ports during a large
	// incremental backlog, while one connection per worker still leaves room
	// for the cursor and watermark queries.
	db.SetMaxOpenConns(maxOpenConnections)
	db.SetMaxIdleConns(maxOpenConnections)

	sq := &PostgresAccessor{db: db}
	return sq, nil
}
