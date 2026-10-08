<p align="center"><img src="build/appicon.png" width="96" alt="Project Library"></p>

# Project Library

App desktop per sfogliare i progetti in `~/Development/{source}/…`, vederne README e stato git e aprirli con l'editor giusto.

Go + [Wails v3](https://v3.wails.io) (beta) per il backend e la finestra, Svelte 5 + Vite per l'interfaccia. Pensata per Windows, macOS e Linux (finora provata solo su Windows). Interfaccia in inglese e italiano, tema chiaro e scuro.
Il riferimento visivo è [mockups/09-colonne-v2.html](mockups/09-colonne-v2.html) (congelato).

## Funzionalità

- **Colonne** con le cartelle annidate come su disco, breadcrumb e logo dell'editor che aprirà ogni progetto.
- **Scheda progetto**: README renderizzato (con le immagini relative), stato git con Fetch, Pull e file modificati, barra dei linguaggi, "Apri con" ed Esplora file.
- **Editor**: 14 editor noti rilevati automaticamente (VS Code, JetBrains, Android Studio…) più quelli aggiunti a mano.
- **Preset e regole**: un preset riconosce un tipo di progetto (Android, Gradle, Node…), una regola gli associa un editor. Le regole sono in ordine di priorità.
- **Ricerca**: palette `Ctrl/⌘ K` nella finestra e ricerca flottante globale (`Win+Ctrl+K`), con i comandi `>`.
- **File**: clona, nuova cartella, rinomina, sposta nel Cestino, "Inizializza progetto" sulle cartelle vuote; la vista si aggiorna da sola quando i file cambiano.
- **Sistema**: tray, avvio con il sistema (no / finestra / solo tray), ripartenza dall'ultima posizione, wizard al primo avvio, esportazione e importazione della configurazione.

## Come funziona

**Classificazione delle cartelle**, in quest'ordine:

1. Eccezione manuale (Impostazioni → Eccezioni): se una cartella è segnata come progetto o come cartella, vale quello.
2. Nomi ignorati (`node_modules`, `vendor`, `target`, `.venv`…) e cartelle nascoste: saltate.
3. Marker (`.git`, `package.json`, `go.mod`, `Cargo.toml`, `pom.xml`, `build.gradle*`, `*.sln`, `README*`…): progetto, la scansione non scende oltre.
4. Almeno un file normale: progetto (opzione attiva di default; `desktop.ini`, `Thumbs.db` e `.DS_Store` non contano).
5. Solo sottocartelle: cartella, la scansione scende. Niente symlink e junction, profondità massima 20.
6. Vuota: cartella vuota, con Nuova cartella, Clona, Inizializza progetto e Segna come progetto.

**Quale editor apre un progetto**: prima quello scelto a mano per quel progetto, poi la prima regola preset → editor che corrisponde, infine l'editor predefinito.

**Eliminare** sposta sempre nel Cestino. Una cartella vuota va subito; il resto chiede conferma; un progetto con modifiche non committate, commit non pushati o senza remote chiede di scriverne il nome.

**git** è quello installato dall'utente, invocato da riga di comando. Se manca, le funzioni git restano disattivate.

## Multipiattaforma

Tutto il codice è comune tranne `internal/platform`, che ha un file per sistema (`_windows.go`, `_darwin.go`, `_linux.go`).

| Funzione | Windows | macOS | Linux |
| --- | --- | --- | --- |
| Cestino | `IFileOperation` (shell) | Finder via `osascript`, altrimenti `~/.Trash` | `gio trash`, altrimenti specifica freedesktop |
| Avvio staccato | `CREATE_NO_WINDOW` + `DETACHED_PROCESS` | `Setsid` | `Setsid` |
| Nomi non validi | `\ / : * ? " < > \|`, nomi riservati, punto o spazio finale | `/` e `:` | `/` |
| Installare git (wizard) | `winget install Git.Git` | `xcode-select --install` | comando della distro da copiare |
| Cartella config | `%APPDATA%\project-library` | `~/Library/Application Support/project-library` | `~/.config/project-library` |

I nomi si confrontano senza distinguere maiuscole su Windows e macOS. Su Linux la tray richiede il supporto StatusNotifier (GNOME ha bisogno di un'estensione).

## Requisiti

- Go 1.24+
- Node 20+ e npm
- CLI di Wails v3: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`, poi `wails3 doctor`
- macOS: Xcode Command Line Tools. Linux: GTK3 e WebKit2GTK (`wails3 doctor` dice quali pacchetti mancano)
- A runtime: `git` (facoltativo)

> Su Windows con Go a 32 bit (`go env GOARCH` = `386`) aggiungi `ARCH=amd64` ai comandi `wails3`.

## Comandi

```sh
wails3 dev                  # sviluppo con hot reload
wails3 build ARCH=amd64     # build di produzione in bin/
wails3 package ARCH=amd64   # installer Windows (NSIS, per utente) in bin/; serve makensis nel PATH
go test ./internal/...
cd frontend && npm run check      # controllo dei tipi Svelte/TypeScript
cd frontend && npm run test:e2e   # test end-to-end (Playwright, backend finto)
```

I binding TypeScript (`frontend/bindings/`) sono generati da `wails3 build`/`wails3 dev` e non sono nel repo.

## Struttura

```text
main.go               finestre (principale e spotlight), tray, scorciatoie di finestra, immagini dei README
internal/
  core/               servizio esposto al frontend
  config/             config.json (default, migrazioni, salvataggio atomico)
  scanner/            albero delle cartelle, classificazione, regole dei launcher
  presets/            catalogo dei preset integrati
  languages/          composizione dei linguaggi di un progetto
  readme/             trova e legge il README, estrae la descrizione
  gitinfo/            stato git e clone tramite il git di sistema
  launcher/           segnaposti e avvio degli editor
  fsops/              crea, rinomina, sposta nel Cestino
  watcher/            fsnotify sulle cartelle di raggruppamento
  platform/           tutto ciò che dipende dal sistema operativo
frontend/
  src/components/     colonne, scheda, palette, spotlight, impostazioni, wizard, dialog
  src/lib/            stato, i18n (en, it), comandi, markdown, menu
  e2e/                test Playwright e backend finto
```

## Dati

La configurazione è in `config.json` nella cartella indicata sopra. Da Impostazioni → Info si esporta, si importa o si reimposta: "Reimposta impostazioni" salva una copia in `config.backup.json`, riporta tutto ai valori iniziali (resta solo la lingua) e riapre il wizard.

## Versioni

Le versioni sono `0.1.x`: ogni funzione o correzione rilasciata incrementa l'ultimo numero. Le release sono su GitHub, con l'installer Windows allegato: le crea `.github/workflows/release.yml` quando si pubblica un tag `v0.1.x` (oppure si avvia a mano da Actions per avere solo l'installer).
