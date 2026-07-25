
# Parameters

Mapserver command line parameters:

## Help
`./mapserver -help`
Shows all available commands


## Version
`./mapserver -version`
Shows the version, architecture and operating system:

```
Mapserver version: 0.0.2
OS: linux
Architecture: amd64
```


## Debug mode
`./mapserver -debug`
Enables the debug mode
It is advisable to pipe the debug output to a file for later inspection:

```
./mapserver -debug > debug.txt
```

## Config
`./mapserver -createconfig`
Creates a config and exits

## Full rerender

`./mapserver -full-rerender`

Resets all initial-render progress, including legacy SQLite progress, and
starts a complete rerender from the lowest configured layer. Existing tile
files and the incremental cursor are retained. Use this once to repair gaps
that were created before safe incremental pagination was installed. Both
`enablerendering` and `enableinitialrendering` must be enabled. The flag resets
progress every time it is supplied, so remove it from persistent service or
container configuration after the repair has started.
