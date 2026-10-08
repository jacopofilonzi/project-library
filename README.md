<p align="center"><img src="build/appicon.png" width="96" alt="Project Library"></p>

# Project Library

Your `~/Development` folder, finally easy to get around.

Project Library is a small desktop app for people with too many repositories. Browse your projects in columns, just like they sit on disk, read their README, see at a glance what changed in git, and open each one in the right editor: IntelliJ for the Gradle project, VS Code for the Node one, without thinking about it. Need something fast? Hit `Win+Ctrl+K` from anywhere, type a few letters and you're in.

> [!NOTE]
> The Windows installer is not code-signed. The first time you run it, SmartScreen shows "Windows protected your PC": choose **More info → Run anyway**. See [Good to know](#good-to-know) for details.

Built with Go + [Wails v3](https://v3.wails.io) (beta) for the backend and the window, Svelte 5 + Vite for the UI. Built for Windows, macOS and Linux (so far tested on Windows only). UI in English and Italian, light and dark theme.

## Features

- **Columns** with nested folders as on disk, breadcrumb and the logo of the editor that will open each project.
- **Project card**: rendered README (with relative images), git status with Fetch, Pull and changed files, language bar, "Open with" and file explorer.
- **Editors**: 14 known editors detected automatically (VS Code, JetBrains, Android Studio…) plus the ones added by hand.
- **Presets and rules**: a preset recognizes a project type (Android, Gradle, Node…), a rule links it to an editor. Rules are in priority order.
- **Search**: `Ctrl/⌘ K` palette in the window and global floating search (`Win+Ctrl+K`), with `>` commands.
- **GitHub and GitLab** (with `gh` or `glab` installed and logged in): clone by picking from your repositories, pull/merge requests, issues and CI in the card, publish a local project that has no remote.
- **Files**: clone, new folder, rename, move (drag a row onto a folder, or "Move to…" in the menu, always with a confirmation), move to the trash, "Initialize project" on empty folders; the view updates by itself when files change.
- **Window**: on Windows the app header is the title bar, with its own minimize / maximize / close buttons that behave like the native ones (snap layouts, drag to the screen edges, double click to maximize). macOS and Linux keep the system title bar.
- **System**: interface size from 80% to 150% (`Ctrl/⌘ +`, `Ctrl/⌘ −`, `Ctrl/⌘ 0` or the slider in Settings → General), tray, start with the system (off / window / tray only), resume from the last location, first-run wizard, configuration export and import.
- **Updates**: at startup the app checks GitHub for a new release; if there is one you get a desktop notification and a small banner with the direct link to the installer. It can be turned off in Settings → About, where you can also check by hand.

## Good to know

**Project status.** This is a personal project, built for my own workflow and shared as is. It runs on [Wails v3](https://v3.wails.io), which is still in beta. It is used daily on Windows; the macOS and Linux code compiles but has never been run, and no package is published for them yet.

**Installing on Windows.** The installer is not code-signed, so Windows SmartScreen shows a "Windows protected your PC" warning the first time. Choose **More info → Run anyway**. The installer works per user: it needs no administrator rights and installs to `%LOCALAPPDATA%\Programs\Project Library`.

**Network and privacy.** The app has no telemetry and collects no data. It goes online only for:

- the update check at startup, a request to the public GitHub API (turn it off in Settings → About);
- `git` fetch, pull and clone, when you ask for them or turn on the background fetch;
- `gh` and `glab`, only if you installed them, with the login you already did.

**What it changes on your disk and accounts.** Browsing is read-only. The app writes only when you ask: new folders, renames, moves, clones, "Initialize project". Deleting always moves to the trash, never deletes permanently. "Publish" is the one action that reaches outside your computer: it creates a real repository on your GitHub or GitLab account and pushes your commits to it.

## How it works

**Folder classification**, in this order:

1. Manual exception (Settings → Exceptions): if a folder is marked as a project or as a folder, that wins.
2. Ignored names (`node_modules`, `vendor`, `target`, `.venv`…) and hidden folders: skipped.
3. Markers (`.git`, `package.json`, `go.mod`, `Cargo.toml`, `pom.xml`, `build.gradle*`, `*.sln`, `README*`…): project, the scan does not go further down.
4. At least one regular file: project (on by default; `desktop.ini`, `Thumbs.db` and `.DS_Store` do not count).
5. Only subfolders: folder, the scan goes down, up to 20 levels. Symlinks and junctions are followed only if enabled in Settings → Scanning.
6. Empty: empty folder, with New folder, Clone, Initialize project and Mark as project.

**Which editor opens a project**: first the one chosen by hand for that project, then the first matching preset → editor rule, finally the default editor.

**Moving** keeps the item's name and works within the watched folders, on the same disk. The editor chosen for a project, its exceptions and its history follow it.

**Per-folder settings** (the editor chosen for a project, the exceptions) are stored in the app's configuration, never inside your projects. They also follow folders renamed or moved outside the app, for example in the file manager: the app recognizes a folder by its file system identity, which does not change when it is renamed or moved on the same disk.

**Deleting** always moves to the trash. An empty folder goes right away; anything else asks for confirmation; a project with uncommitted changes, unpushed commits or no remote asks you to type its name, and so does a folder that contains such a project, at any depth (the dialog lists them).

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
  fsops/              create, rename, move, move to the trash
  watcher/            fsnotify on the grouping folders
  platform/           everything that depends on the operating system
frontend/
  src/components/     columns, card, palette, spotlight, settings, wizard, dialogs
  src/lib/            state, backend access, i18n (en, it), commands, markdown, menus, icons, editor suggestions
  e2e/                Playwright tests and fake backend
```

## Data

The configuration is in `config.json` in the folder listed above. Settings → About exports, imports or resets it: "Reset settings" saves a copy to `config.backup.json`, brings every preference back to the initial values, language included, and reopens the wizard (also from the palette: `> Reset settings`). Updating the app with a new installer keeps the configuration.

## Versions

Versions are `x.y.z`: major, feature, fix. Releases are on GitHub, with the Windows installer attached: `.github/workflows/release.yml` creates them when a `vx.y.z` tag is pushed (or it can be run by hand from Actions to get just the installer). The app's update check reads the latest release from the public GitHub API: drafts and pre-releases are never offered, and the installer must keep the `-installer.exe` name ending.

## License

[MIT](LICENSE) © 2026 Jacopo Filonzi. You can use, modify and redistribute it, also in your own projects, as long as you keep the copyright notice.
