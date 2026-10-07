import en from './en'
import it from './it'

export type Lang = 'en' | 'it'

const dicts = { en, it } as const

const i18n = $state({ lang: 'en' as Lang })

export function setLang(l: string) {
  i18n.lang = l === 'it' ? 'it' : 'en'
  document.documentElement.lang = i18n.lang
}

export function getLang(): Lang {
  return i18n.lang
}

function lookup(dict: unknown, key: string): unknown {
  let cur: unknown = dict
  for (const part of key.split('.')) {
    if (cur && typeof cur === 'object' && part in cur) cur = (cur as Record<string, unknown>)[part]
    else return undefined
  }
  return cur
}

/** Traduce key ("card.modified") sostituendo i parametri {nome}. Ripiega sull'inglese, poi sulla chiave. */
export function t(key: string, vars?: Record<string, string | number>): string {
  let s = lookup(dicts[i18n.lang], key) ?? lookup(en, key)
  if (typeof s !== 'string') return key
  if (vars) s = s.replace(/\{(\w+)\}/g, (m: string, k: string) => (k in vars ? String(vars[k]) : m))
  return s as string
}

/** Come t ma sceglie la variante singolare (key + "1") quando n === 1. */
export function tn(key: string, n: number, vars?: Record<string, string | number>): string {
  return n === 1 ? t(key + '1', { n, ...vars }) : t(key, { n, ...vars })
}

/** Tempo relativo localizzato ("2 h fa" / "2 h ago"). */
export function ago(ms: number): string {
  const s = (Date.now() - ms) / 1000
  if (s < 60) return t('time.now')
  if (s < 3600) return t('time.min', { n: Math.floor(s / 60) })
  if (s < 86400) return t('time.h', { n: Math.floor(s / 3600) })
  const d = Math.floor(s / 86400)
  if (d === 1) return t('time.yesterday')
  if (d < 7) return t('time.days', { n: d })
  return t('time.weeks', { n: Math.floor(d / 7) })
}
