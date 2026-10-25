# Sources and the in-tree registry

Kamps builds its data against one interface:

```go
type Source interface {
    Label() string
    Load(ctx context.Context) (Snapshot, error)
}
```

## Built-in sources

| Name | Package | Notes |
|---|---|---|
| `demo` | `internal/source` | Bundled dataset, no network, download flags are simulated |
| `kulon2` | `internal/kulon2` | Live Moodle source; needs a saved session |

Select one with config or a flag:

```toml
source = "kulon2"
```

```sh
kamps --source demo
```

Register a new in-tree source in `cmd/kamps/sources.go`:

```go
source.Register("mysource", func(opts source.Options) (source.Source, error) {
    return mysource.New(opts.SiteURL), nil
})
```

## Optional capabilities

Sources may implement:

- `source.Downloader` to transfer material bytes.
- `source.SessionReporter` to expose session expiry.
- `source.Capable` to advertise assignments/materials/downloads/session.

The TUI falls back gracefully when an optional interface is absent.

## ID contract

The TUI links materials to assignments by ID:

- assignment: `K<cmid>`
- material: `M<cmid>`
- attachment/folder child: `M<cmid>:<filename>[:n]`

New sources should preserve these shapes (or set `Material.AssignmentID`
explicitly) so attachment links keep working.

Out-of-process JSON plugins are not implemented. The registry keeps the
single-binary model and avoids a runtime ABI.
