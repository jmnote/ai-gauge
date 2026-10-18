# Contributor Notes

- Keep Wails bindings in `internal/app`; keep provider implementations in `internal/providers`.
- Send outbound HTTP requests through `httpclient.Client()` so every request uses the app's User-Agent; don't use `http.DefaultClient`, `http.Get`, or a new `http.Client`.
- Test parsing and conversion with fixture JSON; unit tests must not require live network or CLI calls.
- Keep OS-specific process settings in platform-specific files if cross-platform builds are introduced.

See [docs/development.md](docs/development.md) for build, test, and pre-PR check commands, and
[docs/packaging.md](docs/packaging.md) for MSIX packaging and release commands.
