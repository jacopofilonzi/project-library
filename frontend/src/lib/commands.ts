// Palette commands (activated by typing ">").
// In the floating search, commands that need the main window run there (RunInMain).
import { store, ui, isSpotlight } from './state.svelte'
import { t } from './i18n/index.svelte'
import { lib, errMessage } from './api'

export type Command = {
  id: string
  label: string
  key?: string
  /** needs the main window (dialogs, settings, current folder) */
  main?: boolean
  run: () => void | Promise<void>
}

function canAddHere() {
  const cur = store.current
  return !!cur && cur.kind !== 'root' && !cur.missing
}

export function commands(): Command[] {
  const list: Command[] = []
  if (isSpotlight) list.push({ id: 'show', label: t('cmd.show'), key: store.cfg.hotkey || undefined, run: () => {} , main: true })
  if (!isSpotlight ? canAddHere() : true) {
    list.push({ id: 'newFolder', label: t('cmd.newFolder'), key: store.mod + ' Shift N', main: true, run: () => {
      const cur = store.current
      if (cur && canAddHere()) store.dialog = { kind: 'newFolder', parent: cur.path }
    } })
  }
  list.push(
    { id: 'clone', label: t('cmd.clone'), main: true, run: () => { store.dialog = { kind: 'clone', parent: null } } },
    { id: 'rescan', label: t('cmd.rescan'), run: async () => {
      const tree = await lib.Rescan()
      store.setTree(tree)
      if (!isSpotlight) store.toast(t('settings.scan.rescanned', { n: tree?.count ?? 0 }))
    } },
    { id: 'theme', label: ui.theme === 'dark' ? t('cmd.themeLight') : t('cmd.themeDark'), run: () => { store.save((c) => (c.theme = ui.theme === 'dark' ? 'light' : 'dark')) } },
    { id: 'lang', label: store.cfg.language === 'it' ? 'Switch to English' : "Passa all'italiano", run: () => { store.save((c) => (c.language = c.language === 'it' ? 'en' : 'it')) } },
    { id: 'settings', label: t('cmd.settings'), key: store.mod + ' P', main: true, run: () => store.openSettings() },
  )
  if (!isSpotlight && store.current?.path) {
    const p = store.current.path
    list.push({ id: 'reveal', label: t('cmd.reveal'), key: store.mod + ' E', main: true, run: () => store.reveal(p) })
  }
  list.push(
    { id: 'wizard', label: t('cmd.wizard'), main: true, run: () => { store.settingsOpen = false; store.wizardOpen = true } },
    { id: 'quit', label: t('cmd.quit'), run: () => lib.Quit() },
  )
  return list
}

/** runs a command: in the floating search the "main" ones go to the main window */
export async function runCommand(c: Command) {
  try {
    if (isSpotlight && c.main) {
      await lib.RunInMain(c.id)
      return
    }
    await c.run()
  } catch (e) {
    store.toast(errMessage(e), true)
  }
}

/** run by the main window when the floating search asks for it */
export function runCommandById(id: string) {
  const c = commands().find((x) => x.id === id)
  if (c && id !== 'show') c.run()
}
