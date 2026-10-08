# AGENTS.md

Guide for the agents (and people) working on this repository. Features, behavior rules and differences between systems are described in [README.md](README.md). The UI styles are in `frontend/src/app.css`.

## What it is

Desktop app (Go + Wails v3 beta, Svelte 5 + Vite) to browse the projects in `~/Development/{source}/…`, see their README and git status, and open them with the chosen editor. Targets: Windows, macOS, Linux. UI in English (default) and Italian.

## Commands

```sh
wails3 build ARCH=amd64        # build in bin/ (ARCH=amd64 is needed if `go env GOARCH` is 386)
wails3 dev                     # development with hot reload
wails3 package ARCH=amd64      # per-user NSIS installer in bin/ (needs makensis in the PATH, usually in C:/Program Files (x86)/NSIS)
wails3 generate bindings -clean=true -ts -i   # regenerates frontend/bindings after changing the exposed methods
go test ./internal/...         # with GOARCH=amd64 on Windows: internal/core loads Wails, which does not start as 386
PL_FORGE_LIVE=gh PL_FORGE_REPO=owner/name go test ./internal/forge -run TestLive -v   # live test against the installed gh (or glab), read-only
go vet ./...                   # with GOARCH=amd64 on Windows
cd frontend && npx svelte-check --tsconfig ./tsconfig.json
cd frontend && npm run test:e2e   # Playwright: UI in Chromium with a fake backend (e2e/mock), no Wails, no disk
```

The `internal/platform` package must be checked for the other systems too: `GOOS=darwin go vet $(go list ./internal/... | grep -v /internal/core)` and the same with `GOOS=linux` (`internal/core` and the whole app import Wails, which needs cgo outside Windows and does not cross-compile; Windows instead compiles from any system).

The `wails3` CLI may be in `$(go env GOPATH)/bin/windows_amd64/` if Go is 32-bit.

## Layout

- `main.go`: windows (main and spotlight), tray, window key bindings, middleware that serves the README images.
- `internal/core`: the only service exposed to the frontend (`Library`). Every exported method becomes a binding: helpers for `main` are package functions, not methods.
- `internal/config`: `config.json` (defaults, normalization of missing fields, atomic write). A new field must be added to `Default`, `normalize`, `clone` and, if it holds paths, to `RenamePath`. Per-path settings are keyed by absolute path; `FolderIDs` stores the folder identity (`platform.FileID`) so that `Relink`, run after every scan, moves them when a folder is renamed or moved outside the app.
- `internal/scanner`: folder classification (override → ignore → marker → files → subfolders → empty) and launcher rules.
- `internal/presets`: catalog of the built-in presets (how a project type is recognized). Built-in presets are linked: rules reference them by id and they update with the app; a user preset with a catalog id is ignored (the catalog wins). Rules (`config.Rules`, preset → launcher) are in priority order.
- `internal/languages`: language breakdown of a project (the bar in the card).
- Known editors: the `platform.Editors` table (+ per-system search paths). A new one must also be added to `LauncherIcon.svelte` (logo) and, if needed, to the suggestions in `frontend/src/lib/presets.ts`.
- `internal/platform`: everything that depends on the operating system, in the `_windows.go`, `_darwin.go`, `_linux.go` files. No branching on `runtime.GOOS` outside of it (`core` only reports the OS to the frontend).
- `internal/gitinfo`: uses the `git` installed by the user, never a library. No credential handling.
- `internal/forge`: GitHub and GitLab through the `gh` and `glab` CLIs installed by the user (`gh api`/`glab api`), with their login. Never tokens or libraries. The features that depend on them stay hidden if the CLI is missing or has no account on the host; the exceptions are the hint in the clone dialog and the section in Settings → Git.
- `internal/update`: new release check through the public GitHub releases API (no login). It runs at startup if `CheckUpdates` is on; the desktop notification is sent by `main.go` (`core.OnUpdate`), once per version (`NotifiedVersion`). The installer is found by the end of its name (`platform.UpdateAsset`).
- `internal/fsops`: create, rename, move, move to the trash. Every operation that changes a path must update the config with `RenamePath` (per-project launchers, overrides, history). Errors have a code (`name.badChars`, `exists`…) that the frontend translates.
- `frontend/src/lib/state.svelte.ts`: global state (runes). `frontend/src/lib/i18n/{en,it}.ts`: UI texts; `it.ts` must have the same keys as `en.ts` (the type check verifies it).

## Conventions

- Everything is written in English, except the language packs.
- Every UI text goes through `t()`; always add both the English and the Italian key.
- Deleting means moving to the trash, never deleting permanently.
- Small, step-by-step commits (`feat:`, `fix:`, `chore:`, `docs:`), never a single commit at the end.
- After a change: `go test`, `go vet`, `svelte-check` (0 errors and 0 warnings), `npm run test:e2e` and a build must pass.
- A new method in `internal/core` must also be added to the fake backend `frontend/e2e/mock/library.ts`, otherwise the e2e tests cannot find it.
- Documentation is updated together with the code, in the same commit or right after: a new feature, a rule that changes, a new command or package go into README.md (and here, if they concern developers). It is not piled up for the end.
- UI mockups and design alternatives are never committed: they only serve to choose, the chosen style lives in the code.

## Versions and releases

- Versions are `x.y.z`: `x` major (only when the user decides), `y` feature, `z` fix. A new feature bumps `y` and resets `z` (`1.2.3` → `1.3.0`); a fix bumps `z` (`1.3.0` → `1.3.1`).
- The version is written in 12 files (`build/config.yml`, `build/windows/*`, `build/darwin/Info*.plist`, `build/linux/nfpm/nfpm.yaml`, `internal/core/library.go`, `frontend/package*.json`): change it only with `scripts/set-version.sh x.y.z`, which updates them all and checks that the previous version is left nowhere. Do not use `wails3 update build-assets`: it would regenerate customized files too.
- Every completed feature or fix ends with `scripts/set-version.sh` and a `chore: version x.y.z` commit. Tags and releases are made only when the user asks and include the intermediate versions.
- Release (only when the user asks): annotated `vx.y.z` tag on the current version, push of commits and tag. The tag starts `.github/workflows/release.yml` (runs on Linux and compiles for Windows without cgo): tests, checks, e2e, installer and a GitHub release with notes built from the commits since the previous tag. The workflow stops if the tag does not match the version in `build/config.yml`.
- CI does not run on normal pushes, to save GitHub Actions minutes (the repository is public now, so they are free, but runs still take time). To try the build without releasing: Actions → Release → Run workflow, the installer stays as an artifact for 7 days. Development commits can be pushed freely.

## Known pitfalls

- WebView2 does not pass `Ctrl/⌘ P` to the page: shortcuts that are needed even when the page does not receive them are registered as window `KeyBindings` in `main.go`.
- `autofocus` does not work on elements mounted after load: give the focus with `bind:this` + `$effect`.
- Go slices and maps reach the frontend as `null`: they go through `normalizeConfig` in `frontend/src/lib/api.ts`.
- Custom title bar (Windows only, `platform.CustomTitleBar`): the main window is frameless with `WebView2CompositionHosting`, and the CSS property `--wails-non-client-region` (`caption`, `minimize`, `maximize`, `close`) marks the areas Windows sees as caption and caption buttons. A caption area must not contain clickable elements: it would drag the window instead of clicking. Clicks on the buttons reach the page, so the actions run in `WindowControls.svelte`.
- The interface scale is CSS `zoom` on the page, not the window zoom (Wails keeps the WebView zoom at 100% or more while the app runs). With CSS zoom, viewport units and mouse coordinates are zoomed too: in the CSS divide `vh`/`vw` by `var(--z)`, and divide screen coordinates (`clientX`, `innerWidth`…) by `uiZoom()` before using them as CSS pixels.
- `Win+Ctrl` alone is not a registrable shortcut: a key is always needed (spotlight default: `Super+Ctrl+K`).
