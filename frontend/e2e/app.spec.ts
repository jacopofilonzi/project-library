import { test, expect, type Page } from '@playwright/test'
import type { MockOptions } from './mock/log'

const ROOT = '/home/u/Development'

async function openApp(page: Page, opts: MockOptions = {}, url = '/') {
  await page.addInitScript((o) => (window.__MOCK_OPTIONS = o), opts)
  await page.goto(url)
}

/** riga di una colonna con il nome esatto */
const row = (page: Page, name: string) => page.locator('.row', { has: page.locator('.name', { hasText: new RegExp(`^${name}$`) }) })
const calls = (page: Page) => page.evaluate(() => window.__mock.calls)

test('navigates folders and shows the project card with git and README', async ({ page }) => {
  await openApp(page)
  await row(page, 'github').click()
  await row(page, 'jacopofilonzi').click()
  await row(page, 'TimeTable').click()

  await expect(page.locator('.detail .head h1')).toHaveText('TimeTable')
  await expect(page.locator('.crumbs')).toContainText('jacopofilonzi')
  await expect(page.locator('.git')).toContainText('Up to date with origin')
  await expect(page.locator('.git')).toContainText('github.com/jacopofilonzi/TimeTable')
  await expect(page.locator('.md h2')).toHaveText('Usage')
  await expect(page.getByRole('button', { name: /Open with VS Code/ })).toBeVisible()
})

test('folders with changes show the dot and the list of modified files', async ({ page }) => {
  await openApp(page)
  await row(page, 'github').click()
  await row(page, 'jacopofilonzi').click()
  await expect(row(page, 'NtfyJS').locator('.gd')).toBeVisible()
  await row(page, 'NtfyJS').click()
  await page.locator('.git').getByRole('button', { name: /7 modified files/ }).click()
  await expect(page.locator('.changes .ch')).toHaveCount(7)
  await page.getByRole('button', { name: 'Pull' }).click()
  await expect(page.locator('.toast')).toContainText('Already up to date')
})

test('palette searches the current folder, then everywhere', async ({ page }) => {
  await openApp(page)
  await row(page, 'github').click()
  await row(page, 'jacopofilonzi').click()
  await page.keyboard.press('Control+K')
  const pal = page.locator('.pal')
  await expect(pal.locator('.chip')).toContainText('in jacopofilonzi')
  await page.keyboard.type('bot')
  await expect(pal.locator('.it:not(.glob) .t')).toHaveText(['discord-bot-java'])
  await expect(pal.locator('.it.glob')).toContainText('1 result outside jacopofilonzi')

  await page.keyboard.press('Tab')
  await expect(pal.locator('.chip')).toHaveText('everywhere')
  await expect(pal.locator('.it .t')).toHaveText(['discord-bot-java', 'dity-bot-rs'])

  // ↵ porta al progetto
  await page.keyboard.press('ArrowDown')
  await page.keyboard.press('Enter')
  await expect(pal).toBeHidden()
  await expect(page.locator('.detail .head h1')).toHaveText('dity-bot-rs')
})

test('palette shows recently opened projects everywhere with an empty query', async ({ page }) => {
  await openApp(page)
  await page.keyboard.press('Control+K')
  await expect(page.locator('.pal .sec')).toHaveText('Recently opened')
  await expect(page.locator('.pal .it .t')).toHaveText(['awake'])
})

test('palette commands run with ">"', async ({ page }) => {
  await openApp(page)
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await page.keyboard.press('Control+K')
  await page.keyboard.type('>dark')
  await expect(page.locator('.pal .it .t')).toHaveText(['Switch to dark theme'])
  await page.keyboard.press('Enter')
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
})

test('launcher: content rules and the per-project choice', async ({ page }) => {
  await openApp(page)
  // la regola Java/Gradle sceglie IntelliJ
  await row(page, 'local').click()
  await row(page, 'UNI').click()
  await row(page, 'Ingegneria del Software').click()
  await row(page, 'BuildPatternDemo').click()
  await expect(page.locator('.open .main')).toContainText('Open with IntelliJ IDEA')

  // scelta manuale per un progetto Node
  await page.locator('.crumbs button', { hasText: 'local' }).click()
  await row(page, 'awake').click()
  await expect(page.locator('.open .main')).toContainText('Open with VS Code')
  await page.locator('.open .caret').click()
  await page.getByRole('menuitemradio', { name: 'Always IntelliJ IDEA' }).click()
  await expect(page.locator('.open .main')).toContainText('Open with IntelliJ IDEA')

  await page.locator('.open .main').click()
  expect(await calls(page)).toContainEqual({ fn: 'Open', args: [`${ROOT}/local/awake`, 'intellij'] })
})

test('new folder validates the name and opens the created folder', async ({ page }) => {
  await openApp(page)
  await row(page, 'local').click()
  await page.keyboard.press('Control+Shift+N')
  const dlg = page.locator('.dlg')
  const input = dlg.getByLabel('Name')
  await input.fill('a/b')
  await expect(dlg.locator('.hint.err')).toHaveText('The name cannot contain “/”.')
  await expect(dlg.getByRole('button', { name: 'Create folder' })).toBeDisabled()
  await input.fill('awake')
  await expect(dlg.locator('.hint.err')).toHaveText('An item with this name already exists.')
  await input.fill('scratch')
  await dlg.getByRole('button', { name: 'Create folder' }).click()

  await expect(page.locator('.detail .head h1')).toHaveText('scratch')
  await expect(page.locator('.detail .tag')).toContainText('Empty folder')
})

test('deleting a project without remote needs its name', async ({ page }) => {
  await openApp(page)
  await row(page, 'local').click()
  await row(page, 'awake').click()
  await page.keyboard.press('Delete')
  const dlg = page.locator('.dlg')
  await expect(dlg.locator('.warn')).toContainText('no remote: this is the only copy')
  const del = dlg.getByRole('button', { name: 'Delete' })
  await expect(del).toBeDisabled()
  await page.keyboard.type('awak')
  await expect(del).toBeDisabled()
  await page.keyboard.type('e')
  await del.click()

  await expect(row(page, 'awake')).toHaveCount(0)
  expect(await calls(page)).toContainEqual({ fn: 'Trash', args: [`${ROOT}/local/awake`] })
})

test('an empty folder can be initialized as a project', async ({ page }) => {
  await openApp(page)
  await row(page, 'local').click()
  await row(page, 'Nuova cartella').click()
  await expect(page.locator('.detail .tag')).toContainText('Empty folder')
  await page.getByRole('button', { name: 'Initialize project' }).click()
  await page.locator('.dlg').getByRole('button', { name: 'Initialize' }).click()

  await expect(page.locator('.detail .head h1')).toHaveText('Nuova cartella')
  await expect(page.locator('.readme .bar')).toContainText('README.md')
  expect(await calls(page)).toContainEqual({ fn: 'InitProject', args: [`${ROOT}/local/Nuova cartella`, true, true] })
})

test('settings switch the language', async ({ page }) => {
  await openApp(page)
  await page.getByRole('button', { name: 'Settings' }).click()
  await page.locator('.set select').selectOption('it')
  await expect(page.locator('.set .ph h3')).toHaveText('Generale')
  await page.keyboard.press('Escape')
  await expect(page.locator('.search-btn')).toContainText('Cerca')
})

test('first run shows the setup wizard', async ({ page }) => {
  await openApp(page, { firstRun: true })
  const wiz = page.locator('.wiz')
  await expect(wiz.locator('h2')).toHaveText('Welcome to Project Library')
  await wiz.getByRole('button', { name: 'Italiano' }).click()
  await expect(wiz.locator('h2')).toHaveText('Benvenuto in Project Library')
  await wiz.getByRole('button', { name: 'Avanti' }).click()
  await expect(wiz).toContainText(`${ROOT}`)
  await wiz.getByRole('button', { name: 'Avanti' }).click()
  await expect(wiz.locator('.ed')).toHaveCount(2)
  await expect(wiz.locator('.ed .ok')).toHaveCount(2)
  await wiz.getByRole('button', { name: 'Inizia' }).click()
  await expect(wiz).toBeHidden()
  await expect(row(page, 'github')).toBeVisible()
})

test('the wizard proposes ~/Development only if it exists, otherwise asks to choose', async ({ page }) => {
  await openApp(page, { firstRun: true, missingRoot: true, pickFolder: '/home/u/Projects' })
  const wiz = page.locator('.wiz')
  await wiz.getByRole('button', { name: 'Next' }).click()
  await expect(wiz.locator('.note')).toHaveText(`I could not find ${ROOT}. Choose the folder that contains your projects.`)
  await expect(wiz.locator('.ed')).toHaveCount(0)
  await expect(wiz.getByRole('button', { name: 'Next' })).toBeDisabled()

  await wiz.getByRole('button', { name: 'Choose folder…' }).click()
  await expect(wiz.locator('.ed b')).toHaveText(['/home/u/Projects'])
  await expect(wiz.getByRole('button', { name: 'Next' })).toBeEnabled()
})

test('the wizard offers to install git when it is missing', async ({ page }) => {
  await openApp(page, { firstRun: true, noGit: true })
  const wiz = page.locator('.wiz')
  await expect(wiz.locator('.steps i')).toHaveCount(4)
  for (let i = 0; i < 3; i++) await wiz.getByRole('button', { name: 'Next' }).click()
  await expect(wiz.locator('h2')).toHaveText('git not found')
  await expect(wiz.locator('code.cmd')).toContainText('winget install')
  await expect(wiz.getByRole('button', { name: 'Start' })).toBeDisabled()
  await wiz.getByRole('button', { name: 'Skip' }).click()
  await expect(wiz).toBeHidden()
})

test('reopens where you were', async ({ page }) => {
  await openApp(page, { lastSelected: `${ROOT}/github/jacopofilonzi/NtfyJS` })
  await expect(page.locator('.detail .head h1')).toHaveText('NtfyJS')
  await expect(row(page, 'NtfyJS')).toHaveClass(/on/)
})

test('floating search opens projects and runs commands', async ({ page }) => {
  await openApp(page, {}, '/?view=spotlight')
  const input = page.getByPlaceholder('Search projects, or type > for commands')
  await expect(input).toBeFocused()
  await expect(page.locator('.pal .sec')).toHaveText('Recently opened')
  await page.keyboard.press('Enter')
  await expect.poll(() => calls(page)).toContainEqual({ fn: 'Open', args: [`${ROOT}/local/awake`, 'vscode'] })
  expect(await calls(page)).toContainEqual({ fn: 'HideSpotlight', args: [] })

  // Ctrl ↵ mostra il progetto nella finestra principale
  await input.fill('ntfy')
  await page.keyboard.press('Control+Enter')
  expect(await calls(page)).toContainEqual({ fn: 'ShowInMain', args: [`${ROOT}/github/jacopofilonzi/NtfyJS`] })

  // i comandi che servono alla finestra principale vengono passati a lei
  await input.fill('>settings')
  await page.keyboard.press('Enter')
  await expect.poll(() => calls(page)).toContainEqual({ fn: 'RunInMain', args: ['settings'] })
})
