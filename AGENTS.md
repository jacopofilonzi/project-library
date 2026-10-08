# AGENTS.md

Guida per gli agenti (e le persone) che lavorano su questo repo. Funzionalità, regole di comportamento e differenze tra sistemi sono descritte in [README.md](README.md); il riferimento visivo approvato è [mockups/09-colonne-v2.html](mockups/09-colonne-v2.html) (congelato: non modificarlo).

## Cos'è

App desktop (Go + Wails v3 beta, Svelte 5 + Vite) per sfogliare i progetti in `~/Development/{source}/…`, vederne README e stato git e aprirli con l'editor scelto. Target: Windows, macOS, Linux. Interfaccia in inglese (default) e italiano.

## Comandi

```sh
wails3 build ARCH=amd64        # build in bin/ (ARCH=amd64 serve se `go env GOARCH` è 386)
wails3 dev                     # sviluppo con hot reload
wails3 package ARCH=amd64      # installer NSIS per utente in bin/ (richiede makensis nel PATH, di solito in C:/Program Files (x86)/NSIS)
wails3 generate bindings -clean=true -ts -i   # rigenera frontend/bindings dopo aver cambiato i metodi esposti
go test ./internal/...
PL_FORGE_LIVE=gh PL_FORGE_REPO=owner/nome go test ./internal/forge -run TestLive -v   # prova dal vivo con la gh (o glab) installata, solo letture
go vet ./...                   # con GOARCH=amd64 su Windows
cd frontend && npx svelte-check --tsconfig ./tsconfig.json
cd frontend && npm run test:e2e   # Playwright: interfaccia in Chromium con backend finto (e2e/mock), niente Wails né disco
```

Il pacchetto `internal/platform` va controllato anche per gli altri sistemi: `GOOS=darwin go vet $(go list ./internal/... | grep -v /internal/core)` e lo stesso con `GOOS=linux` (`internal/core` e l'app intera importano Wails, che fuori da Windows richiede cgo e non si cross-compila; per Windows invece si compila da qualunque sistema).

La CLI `wails3` può trovarsi in `$(go env GOPATH)/bin/windows_amd64/` se Go è a 32 bit.

## Struttura

- `main.go`: finestre (principale e spotlight), tray, key binding di finestra, middleware che serve le immagini dei README.
- `internal/core`: unico servizio esposto al frontend (`Library`). Ogni metodo esportato diventa un binding: le funzioni di supporto per `main` sono funzioni di pacchetto, non metodi.
- `internal/config`: `config.json` (default, normalizzazione dei campi mancanti, scrittura atomica). Un campo nuovo va aggiunto a `Default`, `normalize`, `clone` e, se contiene percorsi, a `RenamePath`.
- `internal/scanner`: classificazione delle cartelle (override → ignora → marker → file → sottocartelle → vuota) e regole dei launcher.
- `internal/presets`: catalogo dei preset integrati (come si riconosce un tipo di progetto). I preset integrati sono collegati: le regole li citano per id e si aggiornano con l'app; un preset dell'utente con un id del catalogo viene ignorato (vince il catalogo). Le regole (`config.Rules`, preset → launcher) sono in ordine di priorità.
- `internal/languages`: composizione dei linguaggi di un progetto (barra della scheda).
- Editor noti: tabella `platform.Editors` (+ percorsi di ricerca per sistema). Uno nuovo va aggiunto anche a `LauncherIcon.svelte` (logo) e, se serve, ai consigli di `frontend/src/lib/presets.ts`.
- `internal/platform`: tutto ciò che dipende dal sistema operativo, nei file `_windows.go`, `_darwin.go`, `_linux.go`. Nessun `runtime.GOOS` fuori da qui.
- `internal/gitinfo`: usa il `git` installato dall'utente, mai una libreria. Niente gestione credenziali.
- `internal/forge`: GitHub e GitLab tramite le CLI `gh` e `glab` installate dall'utente (`gh api`/`glab api`), con il loro login. Mai token o librerie. Le funzioni che ne dipendono restano nascoste se la CLI manca o non ha un account sull'host; fanno eccezione il suggerimento nel dialog di clone e la sezione in Impostazioni → Git.
- `internal/fsops`: crea, rinomina, sposta nel Cestino. Gli errori hanno un codice (`name.badChars`, `exists`…) che il frontend traduce.
- `frontend/src/lib/state.svelte.ts`: stato globale (runes). `frontend/src/lib/i18n/{en,it}.ts`: testi; `it.ts` deve avere le stesse chiavi di `en.ts` (lo verifica il type-check).

## Convenzioni

- Commenti e documentazione in italiano; nomi di codice in inglese.
- Ogni testo dell'interfaccia passa da `t()`; aggiungere sempre sia la chiave inglese sia quella italiana.
- Eliminare significa spostare nel Cestino, mai cancellare definitivamente.
- Commit piccoli e per passo (`feat:`, `fix:`, `chore:`, `docs:`), mai un unico commit alla fine.
- Dopo una modifica: `go test`, `go vet`, `svelte-check` (0 errori e 0 avvisi), `npm run test:e2e` e una build devono passare.
- Un metodo nuovo in `internal/core` va aggiunto anche al backend finto `frontend/e2e/mock/library.ts`, altrimenti i test e2e non lo trovano.
- La documentazione si aggiorna insieme al codice, nello stesso commit o subito dopo: una funzione nuova, una regola che cambia, un comando o un pacchetto nuovo vanno riportati in README.md (e qui, se riguardano chi sviluppa). Non si accumula a fine lavoro.

## Versioni e release

- Si resta su `0.1.x`: ogni funzione o correzione rilasciata incrementa l'ultimo numero (`0.1.0` → `0.1.1` → …). Si passa a `0.2` solo se lo decide l'utente.
- La versione è scritta in 11 file (`build/config.yml`, `build/windows/*`, `build/darwin/Info*.plist`, `build/linux/nfpm/nfpm.yaml`, `internal/core/library.go`, `frontend/package*.json`): si cambia solo con `scripts/set-version.sh 0.1.x`, che li aggiorna tutti e controlla che la versione precedente non resti da nessuna parte. Non usare `wails3 update build-assets`: rigenererebbe anche file personalizzati.
- Ogni funzione o correzione completata si chiude con `scripts/set-version.sh` e un commit `chore: version 0.1.x`. Tag e release si fanno solo quando lo chiede l'utente e comprendono le versioni intermedie.
- Rilascio (solo quando lo chiede l'utente): tag annotato `v0.1.x` sulla versione corrente, push di commit e tag. Il tag avvia `.github/workflows/release.yml` (gira su Linux e compila per Windows senza cgo): test, controlli, e2e, installer e release su GitHub con le note ricavate dai commit dal tag precedente. Il workflow si ferma se il tag non coincide con la versione in `build/config.yml`.
- La CI non parte ai push normali, per non consumare i minuti di GitHub Actions (repo privato: 2.000 al mese; per questo si usa Linux, che conta la metà di Windows). Per provare la build senza rilasciare: Actions → Release → Run workflow, l'installer resta come artefatto per 7 giorni. I commit di sviluppo si pushano liberamente.

## Insidie note

- WebView2 non passa `Ctrl/⌘ P` alla pagina: le scorciatoie che servono anche quando la pagina non le riceve si registrano come `KeyBindings` della finestra in `main.go`.
- `autofocus` non funziona sugli elementi montati dopo il caricamento: dare il focus con `bind:this` + `$effect`.
- Le slice e le mappe Go arrivano al frontend come `null`: passano da `normalizeConfig` in `frontend/src/lib/api.ts`.
- `Win+Ctrl` da soli non sono una scorciatoia registrabile: serve sempre un tasto (default spotlight: `Super+Ctrl+K`).
