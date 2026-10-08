// Editor consigliato per i preset del catalogo, usato quando si aggiunge una regola.
import { store } from './state.svelte'
import type { Launcher } from './api'

/** id dei preset integrati → editor noti (builtin) in ordine di preferenza */
const suggestions: Record<string, string[]> = {
  android: ['androidstudio', 'intellij'],
  gradle: ['intellij'],
  maven: ['intellij'],
  intellij: ['intellij'],
  node: ['webstorm', 'vscode'],
  deno: ['webstorm', 'vscode'],
  bun: ['webstorm', 'vscode'],
  go: ['goland', 'vscode'],
  rust: ['rustrover', 'vscode'],
  python: ['pycharm', 'vscode'],
  php: ['phpstorm', 'vscode'],
  ruby: ['rubymine', 'vscode'],
  dotnet: ['rider', 'vscode'],
  cpp: ['clion', 'vscode'],
  flutter: ['androidstudio', 'vscode'],
  swift: ['vscode'],
  unity: ['rider', 'vscode'],
  godot: ['vscode'],
  elixir: ['vscode'],
  vscode: ['vscode'],
}

/**
 * launcher da proporre per un preset: il primo editor consigliato che è abilitato o installato
 * (aggiungendo la regola viene abilitato), altrimenti il predefinito
 */
export function suggestedLauncher(presetId: string): Launcher | undefined {
  const ls = store.cfg.launchers
  for (const b of suggestions[presetId] ?? []) {
    const l = ls.find((x) => x.builtin === b && (x.enabled || !!store.launcherStatus[x.id]))
    if (l) return l
  }
  return store.defaultLauncher ?? ls[0]
}
