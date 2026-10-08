<p align="center"><img src="build/appicon.png" width="96" alt="Project Library"></p>

# Project Library

Your `~/Development` folder, finally easy to get around.

Project Library is a small desktop app for people with too many repositories. Browse your projects in columns, just like they sit on disk, read their README, see at a glance what changed in git, and open each one in the right editor: IntelliJ for the Gradle project, VS Code for the Node one, without thinking about it. Need something fast? Hit `Win+Ctrl+K` from anywhere, type a few letters and you're in.

Built with Go + [Wails v3](https://v3.wails.io) (beta) for the backend and the window, Svelte 5 + Vite for the UI. Built for Windows, macOS and Linux (so far tested on Windows only). UI in English and Italian, light and dark theme.

## Features

- **Columns** with nested folders as on disk, breadcrumb and the logo of the editor that will open each project.
- **Project card**: rendered README (with relative images), git status with Fetch, Pull and changed files, language bar, "Open with" and file explorer.
- **Editors**: 14 known editors detected automatically (VS Code, JetBrains, Android Studio…) plus the ones added by hand.
- **Presets and rules**: a preset recognizes a project type (Android, Gradle, Node…), a rule links it to an editor. Rules are in priority order.
- **Search**: `Ctrl/⌘ K` palette in the window and global floating search (`Win+Ctrl+K`), with `>` commands.
- **GitHub and GitLab** (with `gh` or `glab` installed and logged in): clone by picking from your repositories, pull/merge requests, issues and CI in the card, publish a local project that has no remote.
- **Files**: clone, new folder, rename, move to the trash, "Initialize project" on empty folders; the view updates by itself when files change.
- **System**: tray, start with the system (off / window / tray only), resume from the last location, first-run wizard, configuration export and import.
- **Updates**: at startup the app checks GitHub for a new release; if there is one you get a desktop notification and a small banner with the direct link to the installer. It can be turned off in Settings → About, where you can also check by hand.

## How it works

**Folder classification**, in this order:

1. Manual exception (Settings → Exceptions): if a folder is marked as a project or as a folder, that wins.
2. Ignored names (`node_modules`, `vendor`, `target`, `.venv`…) and hidden folders: skipped.
3. Markers (`.git`, `package.json`, `go.mod`, `Cargo.toml`, `pom.xml`, `build.gradle*`, `*.sln`, `README*`…): project, the scan does not go further down.
4. At least one regular file: project (on by default; `desktop.ini`, `Thumbs.db` and `.DS_Store` do not count).
5. Only subfolders: folder, the scan goes down, up to 20 levels. Symlinks and junctions are followed only if enabled in Settings → Scanning.
6. Empty: empty folder, with New folder, Clone, Initialize project and Mark as project.

**Which editor opens a project**: first the one chosen by hand for that project, then the first matching preset → editor rule, finally the default editor.

**Deleting** always moves to the trash. An empty folder goes right away; anything else asks for confirmation; a project with uncommitted changes, unpushed commits or no remote asks you to type its name.

**git** is the one installed by the user, run from the command line. If it is missing, the git features stay disabled.

**GitHub CLI (`gh`) and GitLab CLI (`glab`)** are optional and are used with the login the user already did (`gh auth login`, `glab auth login`): the app never asks for or stores tokens. Without them the related features do not appear; the clone dialog suggests them, and Settings → Git detects them, offers to install them and shows the accounts. A remote is handled by the CLI that has an account on its host (github.com, gitlab.com or a self-hosted instance).

## Cross-platform

All the code is shared except `internal/platform`, which has one file per system (`_windows.go`, `_darwin.go`, `_linux.go`).

| Feature | Windows | macOS | Linux |
| --- | --- | --- | --- |
| Trash | `SHFileOperationW` (shell, can be undone) | Finder via `osascript`, otherwise `~/.Trash` | `gio trash`, otherwise the freedesktop spec |
| Detached start | `CREATE_NO_WINDOW` + `CREATE_NEW_PROCESS_GROUP` | `Setsid` | `Setsid` |
| Invalid names | `\ / : * ? " < > \|`, reserved names, trailing dot or space | `/` and `:` | `/` |
| Install git (wizard) | `winget install Git.Git` | `xcode-select --install` | distro command to copy |
| Install gh / glab | `winget install GitHub.cli` / `GLab.GLab` | `brew install gh` / `glab` (to copy) | distro package, where there is one |
| Config folder | `%APPDATA%\project-library` | `~/Library/Application Support/project-library` | `~/.config/project-library` |

Names are compared case-insensitively on Windows and macOS. On Linux the tray needs StatusNotifier support (GNOME needs an extension).

## Requirements

- Go 1.25+
- Node 20.19+ or 22.12+ (required by Vite 8) and npm
- Wails v3 CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`, then `wails3 doctor`
- macOS: Xcode Command Line Tools. Linux: GTK3 and WebKit2GTK (`wails3 doctor` tells which packages are missing)
- At runtime: `git` (optional)

> On Windows with 32-bit Go (`go env GOARCH` = `386`) add `ARCH=amd64` to the `wails3` commands.

## Commands

```sh
wails3 dev                  # development with hot reload
wails3 build ARCH=amd64     # production build in bin/
wails3 package ARCH=amd64   # Windows installer (NSIS, per user) in bin/; needs makensis in the PATH
go test ./internal/...
cd frontend && npm run check      # Svelte/TypeScript type check
cd frontend && npm run test:e2e   # end-to-end tests (Playwright, fake backend)
```

The TypeScript bindings (`frontend/bindings/`) are generated by `wails3 build`/`wails3 dev` and are not in the repository.

## Layout

```text
main.go               windows (main and spotlight), tray, window shortcuts, README images
internal/
  core/               service exposed to the frontend
  config/             config.json (defaults, migrations, atomic save)
  scanner/            folder tree, classification, launcher rules
  presets/            catalog of the built-in presets
  languages/          language breakdown of a project
  readme/             finds and reads the README, extracts the description
  gitinfo/            git status and clone through the system git
  update/             new release check on GitHub
  forge/              GitHub and GitLab through gh and glab: accounts, repositories, PRs, CI, creation
  launcher/           placeholders and editor launch
  fsops/              create, rename, move to the trash
  watcher/            fsnotify on the grouping folders
  platform/           everything that depends on the operating system
frontend/
  src/components/     columns, card, palette, spotlight, settings, wizard, dialogs
  src/lib/            state, backend access, i18n (en, it), commands, markdown, menus, icons, editor suggestions
  e2e/                Playwright tests and fake backend
```

## Data

The configuration is in `config.json` in the folder listed above. Settings → About exports, imports or resets it: "Reset settings" saves a copy to `config.backup.json`, brings everything back to the initial values (only the language is kept) and reopens the wizard. Updating the app with a new installer keeps the configuration.

## Versions

Versions are `x.y.z`: major, feature, fix. Releases are on GitHub, with the Windows installer attached: `.github/workflows/release.yml` creates them when a `vx.y.z` tag is pushed (or it can be run by hand from Actions to get just the installer). The app's update check reads the latest release from the public GitHub API: drafts and pre-releases are never offered, and the installer must keep the `-installer.exe` name ending.
