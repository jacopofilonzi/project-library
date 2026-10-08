// Log of the calls to the fake backend, readable from the tests with page.evaluate(() => window.__mock.calls).
export type Call = { fn: string; args: unknown[] }

declare global {
  interface Window {
    __mock: { calls: Call[] }
    __MOCK_OPTIONS?: MockOptions
  }
}

export type MockOptions = {
  /** first-run wizard not completed yet */
  firstRun?: boolean
  /** git not installed */
  noGit?: boolean
  /** last saved location (absolute path of a project or a folder) */
  lastSelected?: string
  language?: 'en' | 'it'
  /** the default root folder (~/Development) does not exist */
  missingRoot?: boolean
  /** path returned by the system folder picker */
  pickFolder?: string
  /** GitHub CLI: missing (default), installed without login, or logged in on github.com */
  forges?: 'none' | 'notLoggedIn' | 'loggedIn'
  /** a newer release is published (the startup check finds it) */
  update?: boolean
}

window.__mock = window.__mock ?? { calls: [] }

export function log(fn: string, ...args: unknown[]) {
  window.__mock.calls.push({ fn, args })
}

export const options: MockOptions = window.__MOCK_OPTIONS ?? {}
