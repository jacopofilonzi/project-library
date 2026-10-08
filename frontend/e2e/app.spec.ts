import { test, expect, type Page } from '@playwright/test'
import type { MockOptions } from './mock/log'

const ROOT = '/home/u/Development'

async function openApp(page: Page, opts: MockOptions = {}, url = '/') {
  await page.addInitScript((o) => (window.__MOCK_OPTIONS = o), opts)
  await page.goto(url)
}

/** column row with the exact name */
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

  // ↵ goes to the project
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
  // the Java/Gradle rule picks IntelliJ
  await row(page, 'local').click()
  await row(page, 'UNI').click()
  await row(page, 'Software Engineering').click()
  await row(page, 'BuildPatternDemo').click()
  await expect(page.locator('.open .main')).toContainText('Open with IntelliJ IDEA')

  // manual choice for a Node project
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
  await row(page, 'New folder').click()
  await expect(page.locator('.detail .tag')).toContainText('Empty folder')
  await page.getByRole('button', { name: 'Initialize project' }).click()
  await page.locator('.dlg').getByRole('button', { name: 'Initialize' }).click()

  await expect(page.locator('.detail .head h1')).toHaveText('New folder')
  await expect(page.locator('.readme .bar')).toContainText('README.md')
  expect(await calls(page)).toContainEqual({ fn: 'InitProject', args: [`${ROOT}/local/New folder`, true, true] })
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
  await wiz.getByRole('button', { name: 'Avanti' }).click()
  await expect(wiz.locator('h2')).toHaveText('Usi più di un IDE?')
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
  await expect(wiz.locator('.steps i')).toHaveCount(5)
  for (let i = 0; i < 3; i++) await wiz.getByRole('button', { name: 'Next' }).click()
  await expect(wiz.locator('h2')).toHaveText('git not found')
  await expect(wiz.locator('code.cmd')).toContainText('winget install')
  await expect(wiz.getByRole('button', { name: 'Next' })).toBeDisabled()
  await wiz.getByRole('button', { name: 'Skip' }).click()
  await wiz.getByRole('button', { name: 'Start' }).click()
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

  // Ctrl ↵ shows the project in the main window
  await input.fill('ntfy')
  await page.keyboard.press('Control+Enter')
  expect(await calls(page)).toContainEqual({ fn: 'ShowInMain', args: [`${ROOT}/github/jacopofilonzi/NtfyJS`] })

  // the commands that need the main window are handed over to it
  await input.fill('>settings')
  await page.keyboard.press('Enter')
  await expect.poll(() => calls(page)).toContainEqual({ fn: 'RunInMain', args: ['settings'] })
})

test('the wizard takes users of several IDEs to the preset library', async ({ page }) => {
  await openApp(page, { firstRun: true })
  const wiz = page.locator('.wiz')
  for (let i = 0; i < 3; i++) await wiz.getByRole('button', { name: 'Next' }).click()
  await wiz.getByRole('button', { name: /Different IDEs depending on the project/ }).click()
  await wiz.getByRole('button', { name: 'Start' }).click()
  await expect(wiz).toBeHidden()
  await expect(page.locator('.set .ph h3')).toHaveText('Launchers')
  // there are already rules (gradle → IntelliJ): the library opens with the button
  await page.getByRole('button', { name: '+ Add rule' }).click()
  await expect(page.locator('.pick .pk', { hasText: 'Java / Kotlin (Gradle)' })).toBeDisabled()
  await expect(page.locator('.pick .pk', { hasText: 'Rust' })).toContainText('VS Code')
})

test('preset rules are ordered by priority', async ({ page }) => {
  await openApp(page)
  await row(page, 'local').click()
  await row(page, 'pocket-app').click()
  // only the Gradle → IntelliJ rule
  await expect(page.locator('.open .main')).toContainText('Open with IntelliJ IDEA')
  await expect(row(page, 'pocket-app').locator('img.lico')).toHaveAttribute('title', 'IntelliJ IDEA')

  await page.getByRole('button', { name: 'Settings' }).click()
  await page.locator('.set nav').getByRole('button', { name: 'Launchers' }).click()
  // Android Studio is installed but not among the launchers yet
  await page.getByRole('button', { name: 'Detect installed editors' }).click()
  await expect(page.locator('.toast')).toContainText('Added (disabled): Android Studio')
  await expect(page.getByRole('checkbox', { name: 'Android Studio' })).not.toBeChecked()

  // the library suggests Android Studio (installed) and adding the rule enables it
  await page.getByRole('button', { name: '+ Add rule' }).click()
  await expect(page.locator('.pick .pk', { hasText: 'Android' }).first()).toContainText('Android Studio')
  await page.locator('.pick .pk', { hasText: 'Android' }).first().click()
  await expect(page.getByRole('checkbox', { name: 'Android Studio' })).toBeChecked()
  const rules = page.locator('.rule:not(.preset) .rb b')
  await expect(rules).toHaveText(['Java / Kotlin (Gradle)', 'Android'])
  // the Gradle rule is above: IntelliJ still wins
  await page.keyboard.press('Escape')
  await expect(page.locator('.open .main')).toContainText('Open with IntelliJ IDEA')

  await page.getByRole('button', { name: 'Settings' }).click()
  await page.locator('.rule:not(.preset)').nth(1).getByRole('button', { name: 'Move up' }).click()
  await expect(rules).toHaveText(['Android', 'Java / Kotlin (Gradle)'])
  await page.keyboard.press('Escape')
  await expect(page.locator('.open .main')).toContainText('Open with Android Studio')
  await expect(row(page, 'pocket-app').locator('img.lico')).toHaveAttribute('title', 'Android Studio')
})

test('built-in presets can be customized into an editable copy', async ({ page }) => {
  await openApp(page)
  await page.getByRole('button', { name: 'Settings' }).click()
  await page.locator('.set nav').getByRole('button', { name: 'Launchers' }).click()
  await page.locator('.rule:not(.preset)').getByRole('button', { name: 'Customize' }).click()
  await expect(page.locator('.rule:not(.preset) .rb b')).toHaveText(['Java / Kotlin (Gradle) (custom)'])
  const copy = page.locator('.rule.preset')
  await expect(copy.locator('.pname')).toHaveValue('Java / Kotlin (Gradle) (custom)')
  await copy.getByPlaceholder('+ add, ↵').fill('pom.xml')
  await page.keyboard.press('Enter')
  await expect(copy.locator('.chips span')).toHaveCount(6)

  // deleting the preset removes its rule too
  await copy.getByRole('button', { name: 'Delete preset (and its rules)' }).click()
  await expect(page.locator('.rule')).toHaveCount(0)
})

test('the project card shows the language overview', async ({ page }) => {
  await openApp(page)
  await row(page, 'local').click()
  await row(page, 'awake').click()
  await expect(page.locator('.langs .lbar span')).toHaveCount(3)
  await expect(page.locator('.langs li')).toHaveText(['TypeScript 70%', 'Svelte 30%', 'Other 0.5%'])
})

test('the configuration can be exported', async ({ page }) => {
  await openApp(page)
  await page.getByRole('button', { name: 'Settings' }).click()
  await page.locator('.set nav').getByRole('button', { name: 'About' }).click()
  await page.getByRole('button', { name: 'Export…' }).click()
  await expect(page.locator('.toast')).toContainText('Configuration exported: /home/u/project-library-config.json')
})

test('resetting the settings asks for confirmation, then runs the wizard', async ({ page }) => {
  await openApp(page)
  await page.getByRole('button', { name: 'Settings' }).click()
  await page.locator('.set nav').getByRole('button', { name: 'About' }).click()
  await page.getByRole('button', { name: 'Reset…' }).click()
  const dlg = page.getByRole('alertdialog', { name: 'Reset all settings?' })
  await expect(dlg).toContainText('/cfg/config.backup.json')
  // Esc closes only the popup, not the settings
  await page.keyboard.press('Escape')
  await expect(dlg).toBeHidden()
  await expect(page.locator('.set')).toBeVisible()
  expect((await calls(page)).some((c) => c.fn === 'ResetConfig')).toBe(false)

  await page.getByRole('button', { name: 'Reset…' }).click()
  await dlg.getByRole('button', { name: 'Reset settings' }).click()
  await expect(page.locator('.set')).toBeHidden()
  await expect(page.locator('.wiz-bg')).toBeVisible()
  expect((await calls(page)).filter((c) => c.fn === 'ResetConfig')).toHaveLength(1)
})

test('settings invite to install GitHub CLI and GitLab CLI', async ({ page }) => {
  await openApp(page)
  await page.getByRole('button', { name: 'Settings' }).click()
  await page.locator('.set nav').getByRole('button', { name: 'Git' }).click()
  const gh = page.locator('.cli', { hasText: 'GitHub CLI' })
  await expect(gh).toContainText('Not installed')
  await expect(page.locator('.cli', { hasText: 'GitLab CLI' })).toContainText('Not installed')
  await gh.getByRole('button', { name: 'Install' }).click()
  await expect(gh).toContainText('The installation started in a new window')
  expect((await calls(page)).some((c) => c.fn === 'RunForgeInstall' && c.args[0] === 'github')).toBe(true)
})

test('settings show how to log in when GitHub CLI has no account', async ({ page }) => {
  await openApp(page, { forges: 'notLoggedIn' })
  await page.getByRole('button', { name: 'Settings' }).click()
  await page.locator('.set nav').getByRole('button', { name: 'Git' }).click()
  const gh = page.locator('.cli', { hasText: 'GitHub CLI' })
  await expect(gh).toContainText('Found: C:/Program Files/GitHub CLI/gh.exe')
  await expect(gh).toContainText('gh auth login')
})

test('without gh or glab the clone dialog only suggests them', async ({ page }) => {
  await openApp(page)
  await page.getByRole('button', { name: 'Clone', exact: true }).click()
  await expect(page.getByRole('tab')).toHaveCount(0)
  await expect(page.locator('.dlg')).toContainText('Tip: with GitHub CLI (gh) or GitLab CLI (glab)')
  await page.getByRole('button', { name: 'Set up in Settings → Git' }).click()
  await expect(page.locator('.set .ph h3')).toHaveText('Git')
})

test('with gh logged in the clone dialog lists your repositories', async ({ page }) => {
  await openApp(page, { forges: 'loggedIn' })
  await page.getByRole('button', { name: 'Clone', exact: true }).click()
  await expect(page.getByRole('tab', { name: 'Your repositories' })).toHaveAttribute('aria-selected', 'true')
  await expect(page.locator('.repo')).toHaveCount(3)
  await page.getByPlaceholder('Search your repositories…').fill('shelly')
  await expect(page.locator('.repo')).toHaveCount(1)
  await page.locator('.repo').click()
  await expect(page.locator('.dlg .final')).toHaveText(`${ROOT}/github/jacopofilonzi/ShellyPlot`)
  await page.locator('.dlg .ft').getByRole('button', { name: 'Clone' }).click()
  await expect(page.locator('.dlg')).toBeHidden()
  const clone = (await calls(page)).find((c) => c.fn === 'Clone')
  expect(clone?.args).toEqual(['git@github.com:jacopofilonzi/ShellyPlot.git', `${ROOT}/github/jacopofilonzi`, 'ShellyPlot'])
})

test('the project card shows pull requests, issues and CI from gh', async ({ page }) => {
  await openApp(page, { forges: 'loggedIn' })
  await row(page, 'github').click()
  await row(page, 'jacopofilonzi').click()
  await row(page, 'TimeTable').click()
  const forge = page.locator('.forge')
  await expect(forge).toContainText('3 issues')
  await expect(forge.locator('.pill.bad')).toContainText('CI failed')
  await forge.getByRole('button', { name: /2 pull requests/ }).click()
  await expect(page.locator('.prs .pr')).toHaveCount(2)
  await expect(page.locator('.prs .pr').nth(1)).toContainText('draft')
  await page.locator('.prs .pr').first().click()
  expect((await calls(page)).some((c) => c.fn === 'OpenURL' && c.args[0] === 'https://github.com/jacopofilonzi/TimeTable/pull/12')).toBe(true)
})

test('without gh the project card has no pull requests', async ({ page }) => {
  await openApp(page)
  await row(page, 'github').click()
  await row(page, 'jacopofilonzi').click()
  await row(page, 'TimeTable').click()
  await expect(page.locator('.git')).toBeVisible()
  await expect(page.locator('.forge')).toHaveCount(0)
  expect((await calls(page)).some((c) => c.fn === 'ForgeInfo')).toBe(false)
})

test('a local project without remote can be published on GitHub', async ({ page }) => {
  await openApp(page, { forges: 'loggedIn' })
  await row(page, 'local').click()
  await row(page, 'awake').click()
  await page.getByRole('button', { name: 'Publish on GitHub…' }).click()
  const dlg = page.locator('.dlg')
  await expect(dlg.locator('h3')).toHaveText('Publish “awake”')
  await expect(dlg.getByLabel('Repository name')).toHaveValue('awake')
  await expect(dlg).toContainText('jacopofilonzi/awake is available')
  // name already used (case-insensitive): no creation
  await dlg.getByLabel('Repository name').fill('ntfyjs')
  await expect(dlg).toContainText('jacopofilonzi/ntfyjs already exists')
  await expect(dlg.getByRole('button', { name: 'Create repository' })).toBeDisabled()
  await dlg.getByLabel('Repository name').fill('awake')
  await dlg.getByLabel('Owner').selectOption('dity-dev')
  await expect(dlg).toContainText('dity-dev/awake is available')
  await dlg.getByLabel('Public').check()
  await dlg.getByRole('button', { name: 'Create repository' }).click()
  await expect(page.locator('.toast')).toContainText('Published on https://github.com/dity-dev/awake')
  const pub = (await calls(page)).find((c) => c.fn === 'Publish')
  expect(pub?.args[0]).toMatchObject({ path: `${ROOT}/local/awake`, kind: 'github', host: 'github.com', name: 'awake', private: false, owner: { name: 'dity-dev' } })
  await expect(page.locator('.git')).toContainText('github.com/dity-dev/awake')
})
