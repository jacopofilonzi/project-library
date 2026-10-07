# Project Library — piano

App desktop per sfogliare i progetti in `~/Development/{source}/…`, leggerne README e stato git e aprirli con l'editor scelto.
Riferimento visivo: [`mockups/09-colonne-v2.html`](mockups/09-colonne-v2.html). Lo stile è approvato: il mockup è congelato.

## Stack

| Livello | Scelta |
|---|---|
| Backend | Go + **Wails v3** (beta, API desktop stabili) |
| Frontend | **Svelte 5 + Vite + TypeScript** |
| Markdown | `marked` + `DOMPurify` (il README è contenuto non fidato) |
| Git | binario `git` installato dall'utente, invocato via CLI |
| Watcher | `fsnotify` |
| i18n | inglese (default) e italiano, file JSON per lingua |
| Piattaforme | Windows, macOS, Linux |
| Module path | `github.com/jacopofilonzi/project-library` |

## Struttura del repo

```
project-library/
├── main.go                     avvio Wails, finestra (MinWidth 920), tray
├── internal/
│   ├── config/                 config.json: load/save, default, migrazioni
│   ├── scanner/                albero cartelle, classificazione progetto/cartella
│   ├── readme/                 trova e legge il README, estrae il primo paragrafo
│   ├── gitinfo/                branch, stato, ahead/behind, ultimo commit, clone
│   ├── launcher/               espansione segnaposti, avvio processi, rilevamento editor
│   ├── fsops/                  crea, rinomina, elimina (Cestino), validazione nomi
│   ├── watcher/                fsnotify sulle cartelle visibili
│   ├── recent/                 cronologia "Apri con" (max 20)
│   └── platform/               tutto ciò che dipende dal sistema operativo (vedi sotto)
│       ├── platform.go         interfaccia comune
│       ├── platform_windows.go
│       ├── platform_darwin.go
│       └── platform_linux.go
├── services/                   servizi esposti al frontend tramite Wails (binding generati)
├── frontend/
│   ├── src/lib/i18n/           en.json, it.json
│   ├── src/lib/components/     Columns, ProjectCard, GitPanel, Readme, OpenWith, Palette, Settings, Wizard, Dialogs
│   └── src/lib/stores/         stato di navigazione, config, tema
└── mockups/                    riferimento UI (congelato)
```

## Multipiattaforma

Il principio: **tutto il codice è comune tranne `internal/platform`**. Lì ogni funzione dipendente dal sistema ha un'implementazione per OS, scelta in compilazione dal suffisso del file (`_windows.go`, `_darwin.go`, `_linux.go`). Il resto dell'app chiama solo l'interfaccia, senza `if runtime.GOOS`.

| Funzione | Windows | macOS | Linux |
|---|---|---|---|
| Mostra nel file manager | `explorer.exe <path>` (exit code 1 ignorato) | `open <path>` | `xdg-open <path>` |
| Sposta nel Cestino | API shell `IFileOperation` con annulla (via `golang.org/x/sys/windows`) | `NSFileManager trashItemAtURL` (piccolo bridge cgo) | `gio trash`, se manca implementazione Go della specifica freedesktop (`~/.local/share/Trash`) |
| Rilevamento VS Code | `PATH`, `%LOCALAPPDATA%\Programs\Microsoft VS Code\bin\code.cmd`, `%ProgramFiles%\Microsoft VS Code\bin\code.cmd` | `PATH`, `/Applications/Visual Studio Code.app/…/bin/code` | `PATH`, `/usr/bin/code`, `/snap/bin/code`, flatpak `com.visualstudio.code` |
| Rilevamento IntelliJ | script di Toolbox `%LOCALAPPDATA%\JetBrains\Toolbox\scripts\idea.cmd`, `%ProgramFiles%\JetBrains\IntelliJ IDEA*\bin\idea64.exe` | `~/Library/Application Support/JetBrains/Toolbox/scripts/idea`, `/Applications/IntelliJ IDEA*.app` | `~/.local/share/JetBrains/Toolbox/scripts/idea`, `/opt/idea*/bin/idea.sh`, snap |
| Avvio processo staccato | `CREATE_NO_WINDOW` + `DETACHED_PROCESS` (niente console per `.cmd`) | `Setsid` | `Setsid` |
| Icona da eseguibile | `SHGetFileInfo` → PNG | `NSWorkspace iconForFile` | `Icon=` del file `.desktop` cercato nel tema icone |
| Avvio con il sistema | registro `HKCU\…\Run` | LaunchAgent in `~/Library/LaunchAgents` | `~/.config/autostart/*.desktop` |
| Nomi non validi | `\ / : * ? " < > \|`, nomi riservati (`CON`, `NUL`…), punto/spazio finale | `/` e `:` | `/` |
| Cartella config | `%APPDATA%\project-library` | `~/Library/Application Support/project-library` | `~/.config/project-library` |
| Installazione git (wizard) | `winget install Git.Git` | `xcode-select --install` | mostra il comando della distro (apt/dnf/pacman) da copiare, senza sudo dall'app |

Altri punti comuni:
- Percorsi solo con `filepath`. Confronto dei nomi case-insensitive su Windows e macOS.
- La validazione dei nomi la fa il backend, il frontend chiede e mostra l'errore.
- Il frontend riceve il sistema operativo da Wails per mostrare `Ctrl` o `⌘`.
- Tray: nativa in Wails v3. Su Linux richiede il supporto StatusNotifier (GNOME ha bisogno di un'estensione).
- Scorciatoia globale: libreria `golang.design/x/hotkey`. Funziona su Windows, macOS e Linux X11, non su Wayland: lì l'opzione viene nascosta.
- Build: macOS va compilato su un Mac. Più avanti GitHub Actions con un job per sistema operativo, ognuno con i test delle sue implementazioni in `platform`.

## Classificazione delle cartelle

Per ogni cartella sotto una root, in quest'ordine:

1. **Override manuale** (Impostazioni → Eccezioni): se l'utente l'ha segnata come progetto o come cartella, vale quello.
2. **Esclusioni**: se il nome è nella lista ignora (`node_modules`, `vendor`, `target`, `.cache`, `.venv`…) o è una cartella nascosta, viene saltata.
3. **Marker → progetto**: contiene almeno uno tra `.git` (cartella o file), `package.json`, `go.mod`, `Cargo.toml`, `pom.xml`, `build.gradle*`, `settings.gradle*`, `gradlew`, `pyproject.toml`, `requirements.txt`, `composer.json`, `*.sln`, `*.csproj`, `CMakeLists.txt`, `Makefile`, `.idea`, `.vscode`, `mise.toml`, `README*`. La scansione non scende oltre.
4. **Contiene almeno un file normale → progetto** (opzione attiva di default). Si ignorano `desktop.ini`, `Thumbs.db` e `.DS_Store`.
5. **Solo sottocartelle → cartella**: la scansione scende al suo interno. Non segue symlink e junction; profondità massima 20.
6. **Vuota → cartella vuota**: viene mostrata, con Apri con, Nuova cartella, Clona e Segna come progetto.

Il tipo e l'icona del linguaggio vengono dai marker. La descrizione è il primo paragrafo del README, altrimenti c'è solo il nome della cartella.

## Funzionalità

- **Navigazione a colonne**: le colonne si nascondono quando lo spazio non basta, minimo 2 colonne più la scheda. Breadcrumb in alto.
- **Scheda progetto**: centrata. In alto nome e percorso, "Apri con" (pulsante doppio se c'è più di un launcher) ed Esplora file. Sotto le info git e il README.
- **README**: Markdown renderizzato e sanificato. Le immagini relative vengono servite dal backend con un asset handler limitato alla cartella del progetto. I link si aprono nel browser. `.rst` e `.txt` come testo semplice.
- **Git** (se disponibile): branch, file modificati, ahead/behind, remote, ultimo commit. Fetch automatico disattivato di default.
- **Ricerca** `Ctrl/⌘ K`: palette centrata, limitata alla cartella corrente, con passaggio a "ovunque". Con "ovunque" e nessun testo mostra gli ultimi 5 progetti aperti.
- **Operazioni sui file**:
  - Clona: da URL https/ssh, propone `github/<owner>`.
  - Nuova cartella.
  - Rinomina.
  - Elimina, sempre nel Cestino:
    - cartella vuota: subito, senza popup;
    - altrimenti: popup di conferma;
    - progetto con modifiche non committate, commit non pushati o senza remote: va scritto il nome per confermare.
- **Watcher**: osserva le cartelle visibili e i progetti aperti, non tutto l'albero, per restare sotto i limiti di inotify e kqueue.
- **Impostazioni** `Ctrl/⌘ P`: pannello modale nella stessa finestra. Sezioni: Generale, Launcher, Scansione, Git, Eccezioni, Scorciatoie, Info. Ogni modifica viene salvata subito.
- **Tema**: chiaro, scuro o di sistema.
- **Lingua**: inglese o italiano, scelta nel wizard e modificabile in Generale.

## Wizard al primo avvio

1. **Lingua**: inglese (preselezionato) o italiano.
2. **Cartelle da osservare**: propone `<home>/Development`. Se non esiste offre di crearla. Se ne possono aggiungere altre.
3. **Editor**: mostra VS Code e IntelliJ con lo stato "trovato in …" o "non trovato". Se non li trova puoi indicare il percorso a mano o disattivarli. Si possono aggiungere altri editor.
4. **Git**: mostrato solo se git non viene trovato. Puoi installarlo (comando per il tuo sistema operativo), indicare un percorso o saltare. Se salti, le funzioni git restano disattivate e l'avviso non compare più.

## Fuori dalla prima versione

Spostamento di cartelle, aggiornamenti automatici dell'app, descrizioni ricavate dai manifest, licenza.

## Fasi

Ogni fase è una serie di commit piccoli, non un commit unico alla fine.

1. **Scheletro**: Wails v3 + Svelte/Vite, finestra, i18n, tema, config.
2. **Scanner + colonne**: classificazione, navigazione, breadcrumb, colonne adattive.
3. **Scheda progetto**: README (con immagini), info git, Apri con, launcher, rilevamento editor.
4. **Ricerca**: palette, cronologia aperture.
5. **Operazioni sui file**: nuova cartella, rinomina, elimina nel Cestino, clona.
6. **Impostazioni + wizard.**
7. **Watcher, tray, scorciatoia globale, avvio con il sistema.**
8. **Rifinitura**: test per OS, build per le tre piattaforme.
