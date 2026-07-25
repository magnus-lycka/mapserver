
# Incremental rendering

Incremental rendering works with the help of the *mtime* column
on the minetest database.

Every insert or update changes the *mtime* column to the current timestamp (with the help of triggers).
This way changes to the blocks can be detected by remembering a compound
cursor consisting of `mtime` and the mapblock position from the last query.


## Table *blocks* (minetest db)

posx	| posy	| posz	| data	| mtime
---	| ---	| ---	| ---	| ---
10	| 11	| 12	| ABC	| 1552977950000
20      | 21    | 22    | 123   | 1552977950010
30      | 31    | 32    | XYZ   | **1552977950020**
40      | 41    | 42    | A12   | 1552977950030
50      | 51    | 52    | B34   | 1552977950040

## Table *settings* (mapserver db)

key		| value
---		| ---
incremental\_cursor\_v1 | `{"version":1,"mtime":1552977950020,"pos":{"x":30,"y":31,"z":32},"position_valid":true}`

## Query example

The following query will return all changed blocks since the last call:

```sql
select posx,posy,posz,data,mtime
from blocks b
where b.mtime > 1552977950020
and b.mtime <= 1552977950080 -- current closed watermark
order by b.mtime asc, b.posx asc, b.posy asc, b.posz asc
limit 1000

```

Additionally it will limit the returned rows so the mapserver can be started and stopped at any time
without processing all new data at once.

If a batch stops partway through one timestamp, the next query first reads the
remaining positions at that timestamp and then continues with later
timestamps. The complete cursor is stored atomically after the raw database
batch has been handled. Layer filtering therefore cannot make pagination
repeat or skip a raw batch.

Existing installations with only `last_mtime` continue from that exclusive
timestamp boundary and write the compound cursor after the first new batch.

## Closed watermark

Rows are only consumed up to a closed upper timestamp watermark. PostgreSQL
caps this watermark below the oldest visible transaction start. The configured
`incrementalrenderingsafetylag` is always applied as an additional safeguard,
including when all transactions are visible. The selected mode is logged.
SQLite applies the time lag and never finalizes the active whole-second
timestamp bucket. The default lag is 5 seconds, based on a measured maximum
PostgreSQL transaction age of 238 ms during a representative large write
operation. It remains configurable because that measurement is not a universal
bound, particularly for SQLite deployments.

This prevents a transaction whose `mtime` was assigned before commit from
appearing behind an already persisted cursor.

## Schedule

Incremental rendering is executed periodically:

* Without pause between calls if there is more data available (catch-up after mapserver downtime)
* With a 5 second pause between calls if there is no new data

Recent rows additionally wait for the configured safety lag before becoming
eligible.

## About realtime

Of course there are delays between placing/removing blocks and the tiles on the mapserver.
The minetest setting **server\_map\_save\_interval** is responsible for the delay to the mapserver (defaults to 5.3 seconds)
Don't try to decrease this value too much on your minetest instance, it has a performance impact!
