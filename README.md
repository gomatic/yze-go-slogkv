# yze-go-slogkv

A [`yze`](https://github.com/gomatic/yze) analyzer (category `logging`) that reports a leveled `slog` call (`slog.Info`/`Warn`/`Error`/`Debug`, or the same methods on a `*slog.Logger`) whose trailing key/value arguments are **unpaired** (odd count) or use a **non-constant-string key**, per the gomatic structured-logging standard.

Calls that pass any `slog.Attr` argument are intentionally **not** checked — mixing pre-built attributes with loose pairs is a valid, harder-to-verify shape, so the analyzer skips it rather than risk a false positive. The `Context`/`Log`/`LogAttrs` variants are out of scope for v1.

- **Rule:** `yze/slogkv`
- **Library:** exports `Analyzer` (a standard `go/analysis` analyzer) and `Registration` for the [`yze`](https://github.com/gomatic/yze) aggregator and [`stickler`](https://github.com/gomatic/stickler) runner.
- **Binary:** `cmd/yze-go-slogkv` runs it standalone (`text`/`-json`, and as a `go vet -vettool`).

Built on the [`go-yze`](https://github.com/gomatic/go-yze) framework.
