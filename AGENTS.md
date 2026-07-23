# AGENTS.md - go-log

Guide for AI agents working in this repository. Pair with `CLAUDE.md` (the working agreement and
hook-enforced rules). Keep this file current when the build, layout, or public API changes.

## What this is

Structured logging for Go services with OpenTelemetry trace correlation. Ships as a Go library
with two parallel public surfaces: a zerolog-returning one (`New`/`Ctx`) for callers that already
depend on zerolog, and a neutral one (`NewLogger`/`LoggerFromContext`, the `Logger` interface,
`Field`/`F`) for callers that must not import `github.com/rs/zerolog` at all. zerolog is the only
backing implementation for both; it is confined to this module and never appears in the neutral
surface's method signatures.

## Using go-log

- Prefer the neutral surface (`log.NewLogger`, `log.LoggerFromContext`, `log.Logger`, `log.F`) for
  any new consumer: it has no zerolog type in any signature, so downstream packages depend only on
  `go-log`.
- The zerolog-returning surface (`log.New`, `log.Ctx`) exists for back-compat and is not going
  away; do not remove or change its signatures.
- Both surfaces read `LOG_LEVEL` (trace/debug/info/warn/error/fatal/panic/disabled, default info)
  and `LOG_FORMAT` (`json` default, `console`/`pretty`, or `both`) from the environment, and both
  attach `trace_id`/`span_id` from the active OpenTelemetry span when one is present in the
  context passed to `Ctx`/`LoggerFromContext`.

## Layout

- `log.go` - the zerolog-returning surface (`New`, `Ctx`) plus the shared `LOG_LEVEL`/`LOG_FORMAT`
  env resolution (`levelFromEnv`, `writer`) both surfaces use.
- `neutral.go` - the neutral surface: `Field`, `F`, the `Logger` interface, `NewLogger`,
  `LoggerFromContext`, and the `neutralLogger` zerolog adapter (zerolog stays internal to this
  file).
- `log_test.go`, `neutral_test.go` - unit tests for each surface.
- `example_neutral_test.go` - a runnable example (`ExampleLogger`) demonstrating the neutral
  surface with zero zerolog references.

## Build, test, lint

- Build: `task build` (`go build ./...`)
- Test: `task test` (`go test ./...`); no external service/fixture required.
- Lint: `task lint` (gofmt check + `golangci-lint run` + `yamllint .`)
- Full local gate: `task ci` (build + `go vet` + test + lint)
- License headers: `task license` (check) / `task license:fix` (inject)

## Conventions and gotchas

- See `CLAUDE.md` for the branch/commit/PR rules; they are enforced by the git hooks in
  `.claude/hooks` (run `bash .claude/hooks/install.sh` once per clone).
- Any change to the neutral `Logger` interface is a public-API change: keep it additive
  (non-breaking) unless the change is explicitly scoped as a major version bump, and update this
  file plus the README when the surface changes.
- `neutralLogger.Ctx` rebuilds from the package-level `Ctx` (same as the zerolog surface) rather
  than merging in the receiver's own `With` fields -- that mirrors `Ctx`'s existing behavior, not a
  bug.
