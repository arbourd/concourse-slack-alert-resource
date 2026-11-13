# AGENTS.md

`concourse-slack-alert-resource` is a Concourse resource type that posts structured Slack alerts from build metadata. Only `out` does work; `check` and `in` are no-ops. Everything ships in one Docker image.

## Commands

```sh
go build ./...                 # build
go test -cpu=1,2 -race ./...   # tests with race detector (what CI runs)
go vet ./...                   # static analysis
golangci-lint run              # lint (CI job)
govulncheck ./...              # known vulnerabilities (CI job)
actionlint                     # lint .github/workflows; run after changing anything there (local only; not run in CI)
docker build -t concourse-slack-alert-resource:dev .
```

## Layout

```
check/main.go, in/main.go   No-op resource steps
out/main.go                 out(): the resource step; message building, previous-build lookup
out/alert.go                Alert and NewAlert: alert type → color, icon, default message
concourse/                  Resource JSON types, BuildMetadata (from env), API Client
slack/slack.go              Message types and Send (retry with backoff)
Dockerfile                  One binary per step at /opt/resource/{check,in,out}
.github/workflows/ci.yml    CI: test, lint, vulncheck; image build, smoke test, publish on main and `v*` tags
.github/renovate.json       Renovate for Dockerfile, Actions, go.mod (runs go mod tidy)
.golangci.yml               golangci-lint config
```

## Data flow

```
main() → out() → NewAlert() → NewBuildMetadata() → buildMessage() → slack.Send()
out() for fixed/broke → previousBuildStatus() → concourse.NewClient() → Client.JobBuild()
```

- `NewAlert` — params override source, which overrides alert-type defaults; either `disable` skips the alert.
- `fixed` and `broke` call the Concourse API for the previous build and send only on a status change. The first build (`BuildName == "1"`) skips the lookup.
- `buildMessage` — `*_file` params are read relative to the build directory via `readLocalFile`, which rejects paths that escape it. An unreadable file logs to stderr and falls back to the plain param.
- `Client` — login differs by Concourse version (legacy token endpoint below 4.0, `/sky/token` below 6.1, split cookies for 5.5–6.4); keep the constraints in `NewClient` intact.
- stdout carries only the JSON `OutResponse`; diagnostics go to stderr or `log.Fatalln`.

## Hard constraints

**Webhook secrecy:** the Slack webhook URL is a secret. Never put it in an error, log or test output; `slack.Send` strips it with `redactURL`. New request code must do the same.

**Error handling:** `fmt.Errorf("context: %w", err)`.

**Dependencies:** avoid adding any; see `go.mod` for what's allowed.

**Go version:** pinned in `go.mod` and in the `Dockerfile` (`golang:<ver>-alpine`); bump both together. CI reads `go.mod`.

**New step:** a new resource step needs a `go build` and a `COPY` in the `Dockerfile`.

**Docs:** new params, source options or alert types go in `README.md`.

## Testing

- Table-driven: `map[string]struct{ ... }` of named cases.
- `t.Setenv()` for env vars; `httptest.NewServer` for Slack and Concourse, never the real network.
- Compare structs with `cmp.Equal`, and report `cmp.Diff` on failure.
- The `image` CI job smoke-tests `/opt/resource/out` against a real webhook (`vars.SLACK_WEBHOOK`); no test should depend on it. `vars.SLACK_WEBHOOK` is deliberately a variable, not a secret (throwaway channel, so PRs can run the smoke test); don't flag it.
