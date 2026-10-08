// Stand-in for @wailsio/runtime in the end-to-end tests (vite --mode e2e).
// In-memory events and fake window/browser; calls are recorded in window.__mock.calls.
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
    // like Wails: asynchronous delivery
    setTimeout(() => listeners.get(name)?.forEach((cb) => cb({ name, data })), 0)
  },
}

export const Window = {
  SetSize: async (w: number, h: number) => log('Window.SetSize', w, h),
  Hide: async () => log('Window.Hide'),
  Show: async () => log('Window.Show'),
  Minimise: async () => log('Window.Minimise'),
  ToggleMaximise: async () => log('Window.ToggleMaximise'),
  Close: async () => log('Window.Close'),
  IsMaximised: async () => false,
}

export const Browser = {
  OpenURL: async (url: string) => log('Browser.OpenURL', url),
}
