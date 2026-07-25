package postgres

const getBlocksByInitialTileQuery = `
select posx,posy,posz,data,mtime
from blocks b
where b.posx >= $1
and b.posy >= $2
and b.posz >= $3
and b.posx <= $4
and b.posy <= $5
and b.posz <= $6
`

const getBlockCountByInitialTileQuery = `
select count(*)
from blocks b
where b.posx >= $1
and b.posy >= $2
and b.posz >= $3
and b.posx <= $4
and b.posy <= $5
and b.posz <= $6
`

const getBlocksByMtimeQuery = `
select posx,posy,posz,data,mtime
from blocks b
where b.mtime > $2 and b.mtime <= $1
order by b.mtime asc, b.posx asc, b.posy asc, b.posz asc
limit $3
`

const getBlocksAtCursorMtimeQuery = `
select posx,posy,posz,data,mtime
from blocks
where mtime = $1
and (
	posx > $2
	or (posx = $2 and posy > $3)
	or (posx = $2 and posy = $3 and posz > $4)
)
order by posx asc, posy asc, posz asc
limit $5
`

const getIncrementalWatermarkQuery = `
select
	floor(extract(epoch from now()) * 1000)::bigint,
	min(floor(extract(epoch from xact_start) * 1000)::bigint)
		filter (where pid <> pg_backend_pid() and usename = current_user),
	min(floor(extract(epoch from xact_start) * 1000)::bigint)
		filter (where pid <> pg_backend_pid()),
	pg_has_role(current_user, 'pg_read_all_stats', 'member')
from pg_stat_activity
where datname = current_database()
`

const countBlocksQuery = `
select count(*) from blocks where mtime >= $1 and mtime <= $2
`

const getTimestampQuery = `
select floor(EXTRACT(EPOCH from now()) * 1000)
`

const getBlockQuery = `
select posx,posy,posz,data,mtime from blocks b
where b.posx = $1
and b.posy = $2
and b.posz = $3
`
