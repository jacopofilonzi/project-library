// Sostituto di @wailsio/runtime per i test end-to-end (vite --mode e2e).
// Eventi in memoria e finestra/browser finti; le chiamate vengono registrate in window.__mock.calls.
import { log } from './log'

type Listener = (ev: { name: string; data: unknown }) => void
const listeners = new Map<string, Set<Listener>>()

export const Events = {
  On(name: string, cb: Listener) {
    if (!listeners.has(name)) listeners.set(name, new Set())
    listeners.get(name)!.add(cb)
    return () => listeners.get(name)?.delete(cb)
  },
  Emit(name: string, data?: unknown) {
    // come Wails: consegna asincrona
    setTimeout(() => listeners.get(name)?.forEach((cb) => cb({ name, data })), 0)
  },
}

export const Window = {
  SetSize: async (w: number, h: number) => log('Window.SetSize', w, h),
  Hide: async () => log('Window.Hide'),
  Show: async () => log('Window.Show'),
}

export const Browser = {
  OpenURL: async (url: string) => log('Browser.OpenURL', url),
}
