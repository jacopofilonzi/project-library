// Punto unico d'accesso al backend (binding generati da Wails).
export * as lib from '../../bindings/github.com/jacopofilonzi/project-library/internal/core/library.js'
export type { ReadmeResult, CloneSuggestion } from '../../bindings/github.com/jacopofilonzi/project-library/internal/core/models.js'
export type { Launcher, Recent } from '../../bindings/github.com/jacopofilonzi/project-library/internal/config/models.js'
import type { AppState as RawAppState } from '../../bindings/github.com/jacopofilonzi/project-library/internal/core/models.js'
import type { Config as RawConfig, Launcher, Recent } from '../../bindings/github.com/jacopofilonzi/project-library/internal/config/models.js'

/** Config con liste e mappe sempre presenti (in Go possono arrivare come null). */
export type Config = Omit<RawConfig, 'roots' | 'launchers' | 'markers' | 'ignore' | 'recent' | 'overrides'> & {
  roots: string[]
  launchers: Launcher[]
  markers: string[]
  ignore: string[]
  recent: Recent[]
  overrides: Record<string, string>
}
export type AppState = Omit<RawAppState, 'config'> & { config: Config }

export function normalizeConfig(c: RawConfig): Config {
  return {
    ...c,
    roots: c.roots ?? [],
    launchers: c.launchers ?? [],
    markers: c.markers ?? [],
    ignore: c.ignore ?? [],
    recent: c.recent ?? [],
    overrides: (c.overrides ?? {}) as Record<string, string>,
  }
}
export function normalizeState(s: RawAppState): AppState {
  return { ...s, config: normalizeConfig(s.config) }
}

/** Evento clone:progress (emesso dal backend, non generato nei binding). */
export type CloneProgress = { id: string; phase: string; percent: number }
export type { Node } from '../../bindings/github.com/jacopofilonzi/project-library/internal/scanner/models.js'
export type { Info as GitInfo } from '../../bindings/github.com/jacopofilonzi/project-library/internal/gitinfo/models.js'
export type { InstallInfo } from '../../bindings/github.com/jacopofilonzi/project-library/internal/platform/models.js'

import { t } from './i18n/index.svelte'

/** Messaggio leggibile da un errore del backend. Gli errori Go hanno la forma "codice: dettaglio". */
export function errMessage(err: unknown): string {
  const raw = typeof err === 'object' && err && 'message' in err ? String((err as { message: unknown }).message) : String(err)
  const i = raw.indexOf(': ')
  const code = i > 0 ? raw.slice(0, i) : raw
  const detail = i > 0 ? raw.slice(i + 2) : ''
  let label: string | undefined
  if (code.startsWith('name.')) label = t(code)
  else if (/^[a-zA-Z]+$/.test(code)) {
    const k = 'errors.' + code
    const tr = t(k)
    if (tr !== k) label = tr
  }
  if (!label) return raw || t('errors.generic')
  return detail ? `${label}: ${detail}` : label
}

/** Codice dell'errore (la parte prima di ": "). */
export function errCode(err: unknown): string {
  const raw = typeof err === 'object' && err && 'message' in err ? String((err as { message: unknown }).message) : String(err)
  const i = raw.indexOf(': ')
  return i > 0 ? raw.slice(0, i) : raw
}
