// Stato globale dell'app (Svelte 5 runes).
import { Events } from '@wailsio/runtime'
import { lib, errMessage, normalizeState, normalizeConfig, type AppState, type Config, type Node, type Launcher, type Preset } from './api'
import { setLang, t } from './i18n/index.svelte'

export type Dialog =
  | { kind: 'newFolder'; parent: string }
  | { kind: 'rename'; path: string; name: string }
  | { kind: 'delete'; node: Node }
  | { kind: 'clone'; parent: string | null }
  | { kind: 'init'; path: string; name: string }

export type CtxItem = { label: string; key?: string; danger?: boolean; run: () => void } | '-'
export type Ctx = { x: number; y: number; items: CtxItem[] }

export type Toast = { id: number; text: string; err?: boolean }

export type Indexed = { node: Node; names: string[]; parent: string[] }

class Store {
  ready = $state(false)
  st = $state<AppState | null>(null)
  tree = $state<Node | null>(null)
  dirty = $state<Record<string, number>>({})
  launcherStatus = $state<Record<string, string>>({})
  /** catalogo dei preset integrati (dall'app) */
  catalog = $state<Preset[]>([])

  /** percorso corrente come nomi a partire dalla radice dell'albero */
  path = $state<string[]>([])
  /** nome del progetto selezionato nella cartella corrente */
  sel = $state<string | null>(null)

  dialog = $state<Dialog | null>(null)
  ctx = $state<Ctx | null>(null)
  paletteOpen = $state(false)
  settingsOpen = $state(false)
  settingsSection = $state('general')
  wizardOpen = $state(false)
  toasts = $state<Toast[]>([])

  get cfg(): Config {
    return this.st!.config
  }
  get os() {
    return this.st?.os ?? 'windows'
  }
  get mod() {
    return this.os === 'darwin' ? '⌘' : 'Ctrl'
  }
  get enabledLaunchers(): Launcher[] {
    return (this.cfg.launchers ?? []).filter((l) => l.enabled)
  }
  get defaultLauncher(): Launcher | undefined {
    const ls = this.enabledLaunchers
    return ls.find((l) => l.id === this.cfg.defaultLauncher) ?? ls[0]
  }

  /**
   * Launcher con cui aprire node, in ordine:
   * 1. scelto per quel progetto (projectLaunchers), se abilitato
   * 2. prima regola che corrisponde ai file del progetto, con launcher abilitato
   * 3. launcher predefinito
   */
  launcherFor(node: Node): { launcher: Launcher | undefined; reason: 'project' | 'rule' | 'default' } {
    const forced = this.enabledLaunchers.find((l) => l.id === this.cfg.projectLaunchers[node.path])
    if (forced) return { launcher: forced, reason: 'project' }
    return this.autoLauncherFor(node)
  }

  /** scelta automatica (regole, poi predefinito), ignorando quella fatta a mano */
  autoLauncherFor(node: Node): { launcher: Launcher | undefined; reason: 'rule' | 'default' } {
    const ls = this.enabledLaunchers
    for (const id of node.ruleLaunchers ?? []) {
      const l = ls.find((x) => x.id === id)
      if (l) return { launcher: l, reason: 'rule' }
    }
    return { launcher: this.defaultLauncher, reason: 'default' }
  }

  /** preset per id, cercato nel catalogo e poi tra quelli dell'utente (come nel backend) */
  presetById(id: string): Preset | undefined {
    return this.catalog.find((p) => p.id === id) ?? this.cfg.presets.find((p) => p.id === id)
  }

  /** sceglie (id) o toglie ('') il launcher fisso di un progetto */
  async setProjectLauncher(node: Node, id: string) {
    const ok = await this.save((c) => {
      if (id) c.projectLaunchers[node.path] = id
      else delete c.projectLaunchers[node.path]
    })
    if (!ok) return
    const l = this.cfg.launchers.find((x) => x.id === id)
    this.toast(l ? t('open.alwaysSaved', { name: node.name, launcher: l.name }) : t('open.autoSaved', { name: node.name }))
  }

  /** etichetta della radice: il nome della cartella o "Radici" con più radici */
  get rootLabel(): string {
    if (!this.tree) return ''
    return this.tree.kind === 'root' ? t('roots') : this.tree.name
  }

  // ---------- albero ----------
  nodeAt(names: string[]): Node | null {
    let n = this.tree
    for (const name of names) {
      n = (n?.children ?? []).find((c) => c?.name === name && c.kind !== 'project') ?? null
      if (!n) return null
    }
    return n
  }
  get current(): Node | null {
    return this.nodeAt(this.path)
  }
  get selected(): Node | null {
    if (!this.sel) return null
    return (this.current?.children ?? []).find((c) => c?.kind === 'project' && c.name === this.sel) ?? null
  }
  /** cosa aprono i launcher: il progetto selezionato o la cartella vuota corrente */
  get target(): Node | null {
    if (this.selected) return this.selected
    const cur = this.current
    if (cur && this.path.length && cur.kind === 'empty' && !cur.missing) return cur
    return null
  }

  /** indice di tutti i progetti: percorso assoluto → nodo e nomi dalla radice */
  index = $derived.by(() => {
    const map = new Map<string, Indexed>()
    const walk = (n: Node, names: string[]) => {
      for (const c of n.children ?? []) {
        if (!c) continue
        if (c.kind === 'project') map.set(c.path, { node: c, names: [...names, c.name], parent: names })
        else walk(c, [...names, c.name])
      }
    }
    if (this.tree) walk(this.tree, [])
    return map
  })

  /** nomi dalla radice per un percorso assoluto di cartella (null se non è nell'albero) */
  namesOf(path: string): string[] | null {
    let found: string[] | null = null
    const walk = (n: Node, names: string[]) => {
      if (found) return
      if (n.path === path) {
        found = names
        return
      }
      for (const c of n.children ?? []) if (c) walk(c, [...names, c.name])
    }
    if (this.tree) walk(this.tree, [])
    return found
  }

  /** dopo un nuovo albero: tiene il percorso valido più lungo */
  private fixPath() {
    let p = [...this.path]
    while (p.length && !this.nodeAt(p)) p.pop()
    if (p.length !== this.path.length) {
      this.path = p
      this.sel = null
    }
    if (this.sel && !this.selected) this.sel = null
  }

  setTree(tree: Node | null) {
    this.tree = tree
    this.fixPath()
  }

  go(names: string[], sel: string | null = null) {
    this.path = names
    this.sel = sel
    this.rememberLocation()
  }

  // salva dove sei (con un piccolo ritardo, la navigazione da tastiera è rapida)
  private locTimer: ReturnType<typeof setTimeout> | undefined
  private rememberLocation() {
    if (isSpotlight || !this.ready) return
    clearTimeout(this.locTimer)
    this.locTimer = setTimeout(() => {
      lib.SetLastLocation(this.current?.path ?? '', this.selected?.path ?? '').catch(() => {})
    }, 600)
  }

  goToPath(path: string) {
    const idx = this.index.get(path)
    if (idx) return this.go(idx.parent, idx.node.name)
    const names = this.namesOf(path)
    if (names) this.go(names)
  }

  // ---------- config ----------
  applyState(raw: Parameters<typeof normalizeState>[0]) {
    const s = normalizeState(raw)
    this.st = s
    setLang(s.config.language)
    applyTheme(s.config.theme)
  }

  /** modifica la config con fn, la salva e applica lo stato restituito */
  async save(fn: (c: Config) => void): Promise<boolean> {
    const next = structuredClone($state.snapshot(this.cfg)) as Config
    fn(next)
    try {
      this.applyState(await lib.SaveConfig(next))
      this.refreshLaunchers()
      return true
    } catch (e) {
      // il salvataggio è avvenuto, ma un effetto (scorciatoia, avvio automatico) è fallito
      this.applyState(await lib.State())
      this.toast(errMessage(e), true)
      return false
    }
  }

  async refreshLaunchers() {
    this.launcherStatus = ((await lib.LauncherStatus()) ?? {}) as Record<string, string>
  }

  // ---------- azioni ----------
  async open(node: Node, launcherId?: string) {
    const l = launcherId ? this.cfg.launchers.find((x) => x.id === launcherId) : this.launcherFor(node).launcher
    if (!l) {
      this.openSettings('launchers')
      return
    }
    try {
      await lib.Open(node.path, l.id)
      if (isSpotlight) lib.HideSpotlight()
      else this.toast(t('open.opened', { launcher: l.name, name: node.name }))
    } catch (e) {
      this.toast(errMessage(e), true)
    }
  }

  async reveal(path: string) {
    try {
      await lib.Reveal(path)
    } catch (e) {
      this.toast(errMessage(e), true)
    }
  }

  async copyPath(path: string) {
    try {
      await navigator.clipboard.writeText(path)
      this.toast(t('ctx.copied', { path }))
    } catch (e) {
      this.toast(errMessage(e), true)
    }
  }

  /** elimina: cartella vuota subito, altrimenti dialog di conferma */
  async askDelete(node: Node) {
    if (node.kind !== 'project' && (await lib.IsEmptyDir(node.path))) {
      await this.trash(node)
      return
    }
    this.dialog = { kind: 'delete', node }
  }

  async trash(node: Node) {
    try {
      await lib.Trash(node.path)
      this.toast(t('dlg.deleted', { name: node.name, trash: this.trashName }))
    } catch (e) {
      this.toast(errMessage(e), true)
    }
  }

  get trashName() {
    return t(this.os === 'windows' ? 'dlg.trashWin' : this.os === 'darwin' ? 'dlg.trashMac' : 'dlg.trashLinux')
  }

  async setOverride(path: string, kind: '' | 'project' | 'dir') {
    try {
      this.applyState(await lib.SetOverride(path, kind))
      this.setTree(await lib.Tree())
    } catch (e) {
      this.toast(errMessage(e), true)
    }
  }

  /** elemento (id) delle impostazioni da mostrare all'apertura */
  settingsAnchor = $state('')

  openSettings(section?: string, anchor = '') {
    this.paletteOpen = false
    this.ctx = null
    this.dialog = null
    if (section) this.settingsSection = section
    this.settingsAnchor = anchor
    this.settingsOpen = true
  }

  toast(text: string, err = false) {
    const id = Date.now() + Math.random()
    this.toasts = [...this.toasts, { id, text, err }]
    setTimeout(() => (this.toasts = this.toasts.filter((x) => x.id !== id)), err ? 5000 : 2200)
  }

  // ---------- avvio ----------
  async init() {
    this.applyState(await lib.State())
    this.setTree(await lib.Tree())
    this.catalog = ((await lib.Presets()) ?? []).map((p) => ({ ...p, patterns: p.patterns ?? [], builtin: true }))
    this.dirty = ((await lib.Dirty()) ?? {}) as Record<string, number>
    this.refreshLaunchers()
    this.wizardOpen = !this.cfg.setupDone
    // git non trovato dopo la configurazione iniziale: avviso una sola volta
    if (this.cfg.setupDone && !this.st?.gitAvailable && !this.cfg.gitWarningShown) {
      this.toast(t('card.gitUnavailable'), true)
      this.save((c) => (c.gitWarningShown = true))
    }
    // riparte da dove eri: progetto selezionato, altrimenti la cartella
    const last = this.cfg.lastSelected || this.cfg.lastPath
    if (!isSpotlight && last) this.goToPath(last)

    Events.On('tree:updated', (ev: { data: Node }) => this.setTree(ev.data))
    Events.On('git:dirty', (ev: { data: Record<string, number> }) => (this.dirty = ev.data ?? {}))
    Events.On('config:updated', (ev: { data: Parameters<typeof normalizeConfig>[0] }) => {
      if (!this.st) return
      const config = normalizeConfig(ev.data)
      const launchersChanged = JSON.stringify(this.st.config.launchers) !== JSON.stringify(config.launchers)
      this.st = { ...this.st, config }
      setLang(config.language)
      applyTheme(config.theme)
      if (launchersChanged) this.refreshLaunchers()
    })
    if (!isSpotlight) {
      // dalla ricerca flottante: "mostra nell'app"
      Events.On('main:goto', (ev: { data: string }) => {
        this.paletteOpen = false
        this.goToPath(ev.data)
      })
      window.addEventListener('focus', () => lib.RefreshDirty())
    }
    this.ready = true
  }
}

/** questa istanza del frontend è la finestra della ricerca flottante */
export const isSpotlight = new URLSearchParams(location.search).get('view') === 'spotlight'

// ---------- tema ----------
const dark = window.matchMedia('(prefers-color-scheme: dark)')
let themeMode = 'system'
/** tema effettivo (reattivo), dopo aver risolto "system" */
export const ui = $state({ theme: 'light' as 'light' | 'dark' })
export function applyTheme(mode: string) {
  themeMode = mode
  ui.theme = mode === 'system' ? (dark.matches ? 'dark' : 'light') : mode === 'dark' ? 'dark' : 'light'
  document.documentElement.dataset.theme = ui.theme
}
dark.addEventListener('change', () => themeMode === 'system' && applyTheme('system'))

export const store = new Store()
