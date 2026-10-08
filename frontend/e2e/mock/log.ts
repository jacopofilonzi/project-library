// Registro delle chiamate al backend finto, leggibile dai test con page.evaluate(() => window.__mock.calls).
export type Call = { fn: string; args: unknown[] }

declare global {
  interface Window {
    __mock: { calls: Call[] }
    __MOCK_OPTIONS?: MockOptions
  }
}

export type MockOptions = {
  /** wizard del primo avvio non ancora completato */
  firstRun?: boolean
  /** git non installato */
  noGit?: boolean
  /** ultima posizione salvata (percorso assoluto di un progetto o di una cartella) */
  lastSelected?: string
  language?: 'en' | 'it'
  /** la cartella radice di default (~/Development) non esiste */
  missingRoot?: boolean
  /** percorso restituito dal selettore di cartelle di sistema */
  pickFolder?: string
  /** GitHub CLI: assente (default), installata senza login o con il login fatto su github.com */
  forges?: 'none' | 'notLoggedIn' | 'loggedIn'
}

window.__mock = window.__mock ?? { calls: [] }

export function log(fn: string, ...args: unknown[]) {
  window.__mock.calls.push({ fn, args })
}

export const options: MockOptions = window.__MOCK_OPTIONS ?? {}
