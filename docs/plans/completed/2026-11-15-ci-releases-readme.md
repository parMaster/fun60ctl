# CI, release binaries and README hotkey docs

**Goal:** every push gets tested and linted, a tag produces downloadable binaries, and the README shows how to install and bind hotkeys.

**Kind of change:** feature

## Intent

Today the only way to get `fun60ctl` is to build it from source. The README says a hotkey tool can call it but shows no rule. No CI runs, so a broken build or lint error is caught only when someone runs it locally. This change adds:

- a CI workflow that runs tests and the linter;
- a release workflow that builds macOS (arm64, amd64) and Windows (amd64) binaries when a `v*` tag is pushed;
- a README install section, a working Karabiner example, and a Bluetooth note.

It works the backlog item `docs/backlog/ci-releases-and-karabiner-example.md` and deletes it.

## Decisions

- **CI runs on `macos-latest`.** go-hid needs cgo everywhere. On Linux it also needs the `libudev` headers, and Linux isn't a target. The repo is public, so macOS minutes are free. Dropped: an Ubuntu runner with `apt install libudev-dev`, which adds a step to test a platform nobody uses.
- **Releases use native runners and `gh release create`.** The macOS runner builds both darwin binaries, because Apple clang targets both CPUs. The Windows runner builds the `.exe` with its preinstalled MinGW gcc. A final job collects the three files and creates the release. Dropped: GoReleaser, which adds a config and needs glue for split cgo builds. Also dropped: one Linux runner with `zig cc`, because zig lacks Apple's IOKit framework, so darwin builds fail.

  ```
  push tag v*
    ├─ macos job   → fun60ctl-darwin-arm64, fun60ctl-darwin-amd64
    ├─ windows job → fun60ctl-windows-amd64.exe
    └─ release job (needs both) → gh release create <tag> + 3 files
  ```

- **The Windows binary ships, marked untested.** The README and the release notes say so until the Windows backlog item lands.
- **The README documents `go install github.com/parMaster/fun60ctl@latest` as the main install path.** Release downloads are the second path, for people without a Go toolchain.
- **The README states that Bluetooth isn't supported.** Over Bluetooth the board is `3151:5027`, and its vendor collection (page `0xFF55`, report ID 6, output and input reports) shares one macOS HID device with the keyboard. Karabiner seizes that device when it modifies the board, so `fun60ctl` can't open it. Keep the note short: one paragraph or one table column, not the probe log.
- **No version flag.** Embedding a version with `-ldflags` is out of scope.

## Constraints / out of scope

- No Go code changes. `main.go`, `layer.go` and `internal/proto` stay as they are.
- No code signing or notarization of the macOS binaries.
- No Windows hardware testing and no PowerToys docs. Those belong to `docs/backlog/windows-build-and-powertoys.md`.
- No Bluetooth support in the code.

## Traps

- `go.mod` declares `go 1.27.3`. Use `setup-go` with `go-version-file: go.mod`. The golangci-lint version must be built with Go ≥ 1.27, or it refuses to run with "the Go language version used to build golangci-lint is lower than the targeted Go version". Pick a golangci-lint release that supports it.
- go-hid links `-framework IOKit -framework CoreFoundation` on darwin (`hid.go` cgo header). Building `GOARCH=amd64` on an arm64 runner needs `CGO_ENABLED=1`. If clang doesn't pick the target CPU from Go, set `CC="clang -arch x86_64"`. Check the result with `file`, which must say `x86_64`.
- The Windows build needs `CGO_ENABLED=1` and a gcc on PATH. Without cgo, go-hid's Windows file doesn't compile, and the build fails with undefined symbols.
- Release downloads on macOS get the quarantine flag, and the binaries are unsigned, so Gatekeeper blocks the first run. The README must give the `xattr -d com.apple.quarantine <file>` fix next to the download instructions.
- Karabiner's `shell_command` runs with a minimal PATH, so the example must use a full path like `/Users/<you>/go/bin/fun60ctl`. `go install` puts the binary in `$(go env GOPATH)/bin`.
- `go install` builds with cgo, so it needs a C compiler. On macOS, that's the Xcode Command Line Tools. Say so next to the install line.
- Locally, `tmp/` sits inside the module and holds `tmp/btprobe/main.go`, so `go test ./...` and the linter pick it up. CI won't see it, because git ignores `tmp/`. Delete `tmp/btprobe` before the final local lint run.
- `golangci-lint` isn't installed on this Mac yet. Install it (`brew install golangci-lint`) before the Wrap-up lint step.

## Definition of Done

- [ ] CI workflow runs `go test ./...` and golangci-lint on every push and pull request to `main` — proof: a green run on GitHub for the pushed commit.
- [x] The code passes golangci-lint with the chosen config — proof: `golangci-lint run ./...` exits 0 locally and in CI.
- [ ] Pushing a `v*` tag creates a GitHub release with `fun60ctl-darwin-arm64`, `fun60ctl-darwin-amd64` and `fun60ctl-windows-amd64.exe` — proof: push a test tag, see the release, then delete the tag and the release if they aren't wanted.
- [ ] The darwin binaries target the right CPU — proof: run `file` on each download. They report `arm64` and `x86_64`.
- [ ] The download for this Mac's CPU (x86_64) runs — proof: after the `xattr` fix, `./fun60ctl-darwin-amd64 actuation` prints the current layer over the cable.
- [x] The release notes and the README say the Windows binary is untested — proof: read both.
- [x] The README has an Install section with `go install`, the C compiler note, release downloads and the quarantine fix — proof: read the README.
- [ ] The README has a working Karabiner example with ctrl+opt+1..4 and a full binary path — proof: paste the README's rule into Karabiner, press ctrl+opt+2 over the cable, and `fun60ctl actuation` prints 2.
- [x] The README says Bluetooth isn't supported and explains the Karabiner conflict — proof: read the README.
- [x] `docs/backlog/ci-releases-and-karabiner-example.md` is deleted with `git rm` in the final commit — proof: `git show --stat HEAD` lists the deletion.

- [x] ➕ A repo-level `.golangci.yml` (v2) exists — proof: without it, golangci-lint picks up a v1 config from `~/go/src` and refuses to run. It enables the standard set plus errorlint, gocritic, misspell, revive, unconvert, gofmt and goimports.
- [x] ➕ `internal/proto/proto_test.go` is gofmt-clean — proof: lint reported a map-literal alignment issue, and `gofmt -w` fixed it.

## Work order

The workflows can only be proven on GitHub, so their proofs come after the push.

1. Write both workflows and the README. Pass the local test and lint runs.
2. Make the single commit and push it. Check the CI run.
3. Push a test tag and check the release, the `file` output and the amd64 run.
4. If a workflow needs a fix, make it a follow-up commit, and don't amend the pushed one.

## Wrap-up

- [x] full test suite passes: `go test ./...`
- [x] linter passes: `golangci-lint run ./...`
- [x] README.md updated (it's part of this change)
- [x] move this plan to `docs/plans/completed/` (`mkdir -p docs/plans/completed && mv docs/plans/2026-11-15-ci-releases-readme.md docs/plans/completed/`)
- [x] single commit: all changes, plan move, and the backlog item deletion

## Post-Completion

- Push the first real tag (for example `v0.1.0`) when you're ready to publish.
