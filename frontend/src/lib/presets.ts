// Suggested editor for the catalog presets, used when a rule is added.
import { store } from './state.svelte'
import type { Launcher } from './api'

/** built-in preset ids → known (builtin) editors in order of preference */
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
 * launcher to suggest for a preset: the first suggested editor that is enabled or installed
 * (adding the rule enables it), otherwise the default one
 */
export function suggestedLauncher(presetId: string): Launcher | undefined {
  const ls = store.cfg.launchers
  for (const b of suggestions[presetId] ?? []) {
    const l = ls.find((x) => x.builtin === b && (x.enabled || !!store.launcherStatus[x.id]))
    if (l) return l
  }
  return store.defaultLauncher ?? ls[0]
}
