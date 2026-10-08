# AGENTS.md

Guida per gli agenti (e le persone) che lavorano su questo repo. Le decisioni di prodotto sono in [PLAN.md](PLAN.md); il riferimento visivo approvato è [mockups/09-colonne-v2.html](mockups/09-colonne-v2.html) (congelato: non modificarlo).

## Cos'è

App desktop (Go + Wails v3 beta, Svelte 5 + Vite) per sfogliare i progetti in `~/Development/{source}/…`, vederne README e stato git e aprirli con l'editor scelto. Target: Windows, macOS, Linux. Interfaccia in inglese (default) e italiano.

## Comandi

```sh
wails3 build ARCH=amd64        # build in bin/ (ARCH=amd64 serve se `go env GOARCH` è 386)
wails3 dev                     # sviluppo con hot reload
wails3 package ARCH=amd64      # installer NSIS per utente in bin/ (richiede makensis, es. "C:Program Files (x86)NSIS")
wails3 generate bindings -clean=true -ts -i   # rigenera frontend/bindings dopo aver cambiato i metodi esposti
go test ./internal/...
go vet ./...                   # con GOARCH=amd64 su Windows
cd frontend && npx svelte-check --tsconfig ./tsconfig.json
```

Il pacchetto `internal/platform` va controllato anche per gli altri sistemi: `GOOS=darwin go vet ./internal/...` e `GOOS=linux go vet ./internal/...` (l'app intera richiede cgo fuori da Windows e non si cross-compila).

La CLI `wails3` può trovarsi in `$(go env GOPATH)/bin/windows_amd64/` se Go è a 32 bit.

## Struttura

- `main.go`: finestre (principale e spotlight), tray, key binding di finestra, middleware che serve le immagini dei README.
- `internal/core`: unico servizio esposto al frontend (`Library`). Ogni metodo esportato diventa un binding: le funzioni di supporto per `main` sono funzioni di pacchetto, non metodi.
- `internal/config`: `config.json` (default, normalizzazione dei campi mancanti, scrittura atomica). Un campo nuovo va aggiunto a `Default`, `normalize`, `clone` e, se contiene percorsi, a `RenamePath`.
- `internal/scanner`: classificazione delle cartelle (override → ignora → marker → file → sottocartelle → vuota) e regole dei launcher.
- `internal/platform`: tutto ciò che dipende dal sistema operativo, nei file `_windows.go`, `_darwin.go`, `_linux.go`. Nessun `runtime.GOOS` fuori da qui.
- `internal/gitinfo`: usa il `git` installato dall'utente, mai una libreria. Niente gestione credenziali.
- `internal/fsops`: crea, rinomina, sposta nel Cestino. Gli errori hanno un codice (`name.badChars`, `exists`…) che il frontend traduce.
- `frontend/src/lib/state.svelte.ts`: stato globale (runes). `frontend/src/lib/i18n/{en,it}.ts`: testi; `it.ts` deve avere le stesse chiavi di `en.ts` (lo verifica il type-check).

## Convenzioni

- Commenti e documentazione in italiano; nomi di codice in inglese.
- Ogni testo dell'interfaccia passa da `t()`; aggiungere sempre sia la chiave inglese sia quella italiana.
- Eliminare significa spostare nel Cestino, mai cancellare definitivamente.
- Commit piccoli e per passo (`feat:`, `fix:`, `chore:`, `docs:`), mai un unico commit alla fine.
- Dopo una modifica: `go test`, `go vet`, `svelte-check` e una build devono passare.

## Insidie note

- WebView2 non passa `Ctrl/⌘ P` alla pagina: le scorciatoie che servono anche quando la pagina non le riceve si registrano come `KeyBindings` della finestra in `main.go`.
- `autofocus` non funziona sugli elementi montati dopo il caricamento: dare il focus con `bind:this` + `$effect`.
- Le slice e le mappe Go arrivano al frontend come `null`: passano da `normalizeConfig` in `frontend/src/lib/api.ts`.
- `Win+Ctrl` da soli non sono una scorciatoia registrabile: serve sempre un tasto (default spotlight: `Super+Ctrl+K`).
