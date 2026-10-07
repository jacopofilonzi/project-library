# play-evons-gg

Web game client for the Evons CCG. The backend
([`evons-game-backend`](../evons-game-backend)) is the sole authority on the
rules; this is **only the frontend**. It renders the filtered view and the
`legalActions` the server sends, and echoes the chosen action back.

Next.js (App Router) + React + TypeScript, **fully static export** (`output:
'export'`). All auth, deck and game calls are client-side to the backend; the
in-match logic is client-side too, but it never computes rules — it follows the
server's `legalActions`.

## Run

```bash
npm install
npm run dev        # http://localhost:3002
```

By default this app talks to the deployed backend at `https://evons.dity.me`
(REST) and `wss://evons.dity.me/ws` (WebSocket) in every environment. To run
against a local backend (see ../evons-game-backend: `npm run db:up`,
`npm run db:migrate`, `npm run dev` on :8080), set `NEXT_PUBLIC_API_BASE` /
`NEXT_PUBLIC_WS_BASE` (see `.env.example`).

To try a 2-player match, open two browser windows and sign in as two users.

```bash
npm run typecheck
npm run build      # static export to ./out
```

## Structure

```
app/
  page.tsx              Beta landing (all cards free & unlimited)
  login/                Login + register (REST → JWT)
  dashboard/            Home: welcome carousel + shortcuts
  dashboard/matches/    Lobby (quick match / create / join + open matches)
  dashboard/decks/      Deck slots; opens the builder on ?slot=
  play/                 Game table (client-only)
components/
  auth/             AuthProvider (token), AuthForm, AuthGuard
  layout/           AppShell (guard + WS provider + sidebar), Sidebar (nav)
  dashboard/        DashboardHome + HomeCarousel, MatchesContent (lobby),
                    DecksContent, DeckSlots
  deck/             DeckEditor (single deck, saves to DB)
  game/             GameConnectionProvider (owns the WebSocket + reduces STATE),
                    GameTable, Board, LeftSidebar, RightPanel, ActionDialog,
                    CardTile, PlayGate
lib/
  api.ts            REST: auth + decks
  ws.ts             GameSocket: WS transport + reconnect
  auth.ts           Token/identity persistence (localStorage)
  board.ts          name ↔ coord conversion (the single source of truth)
  actions.ts        legalActions interpreters
  catalog.ts        card lookup/render (set-00.json for now)
  deck.ts           deck shape + client-side validation
types/
