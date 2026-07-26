# Mapblocks and render layers

Mapserver divides a world into independently selectable vertical render layers.
Layers are configured in the `layers` array in `mapserver.json`:

```json
"layers": [
  {
    "id": 0,
    "name": "Surface",
    "from": -1,
    "to": 24
  },
  {
    "id": 1,
    "name": "Shallow underground",
    "from": -8,
    "to": -2
  }
]
```

`from` and `to` are inclusive **mapblock Y coordinates**, not node Y
coordinates. A mapblock is 16 nodes high. Mapblock Y coordinate `B` contains
node heights:

```text
16 * B through 16 * B + 15, inclusive
```

Consequently, a layer from mapblock `F` through mapblock `T` includes node
heights:

```text
16 * F through 16 * T + 15, inclusive
```

Negative coordinates follow the same formula. In particular, mapblock `-1`
contains node heights `-16` through `-1`; it does not contain node height `0`.

## Common boundaries

| Mapblock Y | Inclusive node Y range |
|-----------:|-----------------------:|
| -32 | -512 through -497 |
| -16 | -256 through -241 |
| -9 | -144 through -129 |
| -8 | -128 through -113 |
| -4 | -64 through -49 |
| -3 | -48 through -33 |
| -2 | -32 through -17 |
| -1 | -16 through -1 |
| 0 | 0 through 15 |
| 1 | 16 through 31 |
| 10 | 160 through 175 |
| 11 | 176 through 191 |
| 12 | 192 through 207 |
| 24 | 384 through 399 |

Some example complete layer ranges are:

| Layer mapblock range | Inclusive node Y range |
|---------------------:|-----------------------:|
| `-1` through `24` | -16 through 399 |
| `-8` through `-2` | -128 through -17 |
| `-32` through `-9` | -512 through -129 |

For example, a summit at node Y=203 is in mapblock Y=12. It is excluded from
a layer whose `to` value is 10, because that layer stops at node Y=175.

## What a layer displays

For each horizontal node position, Mapserver scans downward from the top of the
layer and draws the first node for which it has a color. Air and nodes without a
color are skipped.

This makes layers useful for surface terrain, floating structures, open
caverns, and vertically separated areas. It is not a true horizontal
cross-section:

- empty air above a structure does not hide it;
- a solid roof or natural stone above an underground structure does hide it;
- an enclosed tunnel normally appears as its roof rather than its interior;
- the vertical cut can only be placed at a 16-node mapblock boundary.

To display an underground level, place the layer's upper boundary close to the
desired cut height. Existing structures may need to be designed around these
boundaries if their interiors, rather than their roofs, should be visible.

## Layer constraints

Layer ranges should not overlap. A changed mapblock is assigned to the first
configured layer whose range contains its Y coordinate. With overlapping
ranges, later layers may therefore not receive incremental updates. Gaps are
allowed, but mapblocks in a gap are not rendered by any layer.

Each layer must have a unique, stable `id`. The ID is part of its tile-storage
path, so do not reuse an old ID for a different layer while retaining its tiles.

Changing layer IDs or vertical ranges invalidates existing tiles. After such a
change, perform one complete rerender:

```sh
./mapserver --full-rerender
```

After it completes, start Mapserver normally without `--full-rerender`.
