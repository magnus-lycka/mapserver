
# Repairing incomplete rendering

To repair historical gaps, use the supported full-rerender operation:

```sh
./mapserver -full-rerender
```

This resets every initial-render progress component and starts from the lowest
configured layer. It does not delete tiles or reset the incremental cursor.
Setting only `initial_run=true` or editing `last_mtime` is insufficient and may
skip layers or conflict with the compound incremental cursor.
