<p align="center"><img src="build/appicon.png" width="96" alt="Project Library"></p>

# Project Library

App desktop per sfogliare i progetti in `~/Development/{source}/…`, vederne README e stato git e aprirli con l'editor scelto.

Go + [Wails v3](https://v3.wails.io) (beta) per il backend e la finestra, Svelte 5 + Vite per l'interfaccia. Funziona su Windows, macOS e Linux.
Il piano completo e le decisioni prese sono in [PLAN.md](PLAN.md); il riferimento visivo è [mockups/09-colonne-v2.html](mockups/09-colonne-v2.html).

## Requisiti

- Go 1.24+
- Node 20+ e npm
- CLI di Wails v3: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`, poi `wails3 doctor`
- macOS: Xcode Command Line Tools. Linux: GTK3 e WebKit2GTK (`wails3 doctor` dice quali pacchetti mancano)
- A runtime: `git` installato (facoltativo, senza le funzioni git restano disattivate)

> Su Windows con Go a 32 bit (`go env GOARCH` = `386`) compila a 64 bit con `GOARCH=amd64`.

## Comandi

```sh
wails3 dev            # sviluppo con hot reload
wails3 build          # build di produzione in bin/
wails3 package ARCH=amd64   # installer Windows (NSIS, per utente) in bin/; serve makensis nel PATH
go test ./internal/...
cd frontend && npm run check   # controllo dei tipi Svelte/TypeScript
```

I binding TypeScript (`frontend/bindings/`) sono generati da `wails3 build`/`wails3 dev` e non sono nel repo.

## Struttura

```
main.go               finestra, tray, scorciatoie di finestra, immagini dei README
internal/
  config/             config.json (default, caricamento, salvataggio atomico)
  scanner/            albero delle cartelle e classificazione progetto/cartella
  readme/             trova e legge il README, estrae la descrizione
  gitinfo/            stato git e clone tramite il git di sistema
  launcher/           segnaposti e avvio degli editor
  fsops/              crea, rinomina, sposta nel Cestino
  watcher/            fsnotify sulle cartelle di raggruppamento
  platform/           tutto ciò che dipende dal sistema operativo (_windows, _darwin, _linux)
  core/               servizio esposto al frontend
frontend/src/
  components/         colonne, scheda progetto, palette, impostazioni, wizard, dialog
  lib/                stato, i18n (en, it), markdown, menu
```

## Dati

La configurazione è in `%APPDATA%\project-library\config.json` (Windows), `~/Library/Application Support/project-library/config.json` (macOS) o `~/.config/project-library/config.json` (Linux). Il wizard del primo avvio si può ripetere da Impostazioni → Info.
