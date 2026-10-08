// Fake backend for the end-to-end tests: same functions as the internal/core (Library) bindings,
// with a small in-memory project tree that really creates, renames and deletes.
import { Events } from './runtime'
import { log, options } from './log'

const ROOT = '/home/u/Development'
const join = (...p: string[]) => p.join('/')
const base = (p: string) => p.slice(p.lastIndexOf('/') + 1)
const parentOf = (p: string) => p.slice(0, p.lastIndexOf('/'))

// ---------- fake file system ----------
type FNode = { name: string; kind: 'dir' | 'project' | 'empty'; lang?: string; desc?: string; hasGit?: boolean; files?: string[]; children?: FNode[] }

const proj = (name: string, lang: string, files: string[], extra: Partial<FNode> = {}): FNode => ({ name, kind: 'project', lang, files, ...extra })
const dir = (name: string, children: FNode[]): FNode => ({ name, kind: 'dir', children })

const fs: FNode = dir('Development', [
  dir('github', [
    dir('curishi', [proj('play.evons.gg', 'Node', ['package.json', 'README.md', '.git'], { hasGit: true, desc: 'Web game client for the Evons CCG.' })]),
    dir('jacopofilonzi', [
      proj('discord-bot-java', 'Java', ['settings.gradle.kts', 'gradlew', '.git'], { hasGit: true }),
      proj('NtfyJS', 'Node', ['package.json', 'README.md', '.git'], { hasGit: true, desc: 'An ntfy client for Javascript and Typescript' }),
      proj('TimeTable', '', ['README.md', '.git'], { hasGit: true, desc: "Subscribe to your university's lesson timetable." }),
    ]),
  ]),
  dir('local', [
    { name: 'New folder', kind: 'empty' },
    dir('UNI', [dir('Software Engineering', [proj('BuildPatternDemo', 'Java', ['build.gradle.kts', 'gradlew'])])]),
    proj('awake', 'Node', ['package.json', 'README.md', '.git'], { hasGit: true, desc: 'Self-hosted Wake-on-LAN over the internet.' }),
    proj('dity-bot-rs', 'Rust', ['Cargo.toml', '.git'], { hasGit: true }),
    proj('pocket-app', 'Kotlin', ['app/src/main/AndroidManifest.xml', 'build.gradle.kts', 'gradlew']),
  ]),
])

function find(path: string): { node: FNode; parent: FNode | null } | null {
  if (path === ROOT) return { node: fs, parent: null }
  if (!path.startsWith(ROOT + '/')) return null
  let node = fs
  let parent: FNode | null = null
  for (const seg of path.slice(ROOT.length + 1).split('/')) {
    const next = node.children?.find((c) => c.name === seg)
    if (!next) return null
    parent = node
    node = next
  }
  return { node, parent }
}

const globRe = (g: string) => new RegExp('^' + g.replace(/[.+^${}()|[\]\\]/g, '\\$&').replace(/\*/g, '.*').replace(/\?/g, '.') + '$', 'i')

function toTree(n: FNode, path: string): any {
  if (n.kind === 'project') {
    // like scanner.MatchRules: launchers of the matching rules, in order and without duplicates
    const ruleLaunchers: string[] = []
    for (const r of config.rules ?? []) {
      const p = findPreset(r.preset)
      if (p && !ruleLaunchers.includes(r.launcher) && p.patterns.some((g: string) => (n.files ?? []).some((f) => globRe(g).test(f)))) ruleLaunchers.push(r.launcher)
    }
    return { name: n.name, path, kind: 'project', lang: n.lang, desc: n.desc, hasGit: n.hasGit, count: 1, ruleLaunchers }
  }
  const children = (n.children ?? [])
    .map((c) => toTree(c, join(path, c.name)))
    .sort((a, b) => (a.kind === 'project') === (b.kind === 'project') ? a.name.toLowerCase().localeCompare(b.name.toLowerCase()) : a.kind === 'project' ? 1 : -1)
  const kind = n.kind === 'empty' || !children.length ? (n === fs ? 'dir' : 'empty') : 'dir'
  return { name: n.name, path, kind, count: children.reduce((s: number, c: any) => s + c.count, 0), children }
}

// ---------- presets (part of the internal/presets catalog) ----------
const catalog = [
  { id: 'android', name: 'Android', patterns: ['app/src/main/AndroidManifest.xml', 'src/main/AndroidManifest.xml', 'AndroidManifest.xml'] },
  { id: 'gradle', name: 'Java / Kotlin (Gradle)', patterns: ['build.gradle', 'build.gradle.kts', 'settings.gradle', 'settings.gradle.kts', 'gradlew'] },
  { id: 'maven', name: 'Java (Maven)', patterns: ['pom.xml', 'mvnw'] },
  { id: 'node', name: 'Node.js', patterns: ['package.json'] },
  { id: 'rust', name: 'Rust', patterns: ['Cargo.toml'] },
]
const findPreset = (id: string) => catalog.find((p) => p.id === id) ?? (config.presets ?? []).find((p: any) => p.id === id)

// known editors "installed" on the fake machine, besides VS Code and IntelliJ
const installed: Record<string, string> = { vscode: 'C:/VSCode/Code.exe', intellij: 'C:/JetBrains/idea64.exe', androidstudio: 'C:/Android/studio64.exe' }
const knownNames: Record<string, string> = { androidstudio: 'Android Studio' }

// ---------- fake git ----------
const gitData: Record<string, any> = {
  [join(ROOT, 'github/jacopofilonzi/TimeTable')]: { isRepo: true, branch: 'main', remote: 'git@github.com:jacopofilonzi/TimeTable.git', remoteWeb: 'https://github.com/jacopofilonzi/TimeTable', hasUpstream: true, ahead: 0, behind: 0, dirty: 0, commit: { hash: 'af8c550', subject: 'Feat: usage tracking', author: 'Filonzi Jacopo', time: 1791000000 } },
  [join(ROOT, 'github/jacopofilonzi/NtfyJS')]: { isRepo: true, branch: 'main', remote: 'https://github.com/jacopofilonzi/NtfyJS', remoteWeb: 'https://github.com/jacopofilonzi/NtfyJS', hasUpstream: true, ahead: 1, behind: 0, dirty: 7, commit: { hash: '5428c14', subject: 'Blanked gitignore', author: 'Filonzi Jacopo', time: 1788000000 } },
  [join(ROOT, 'local/awake')]: { isRepo: true, branch: 'master', remote: '', remoteWeb: '', hasUpstream: false, ahead: 0, behind: 0, dirty: 0, commit: { hash: '06e157a', subject: 'Initial Awake implementation', author: 'Filonzi Jacopo', time: 1790000000 } },
}
const gitOf = (p: string) => gitData[p] ?? { isRepo: true, branch: 'main', remote: 'git@github.com:x/y.git', remoteWeb: 'https://github.com/x/y', hasUpstream: true, ahead: 0, behind: 0, dirty: 0, commit: null }

// ---------- config ----------
let config: any = {
  version: 3,
  language: options.language ?? 'en',
  theme: 'light',
  roots: [ROOT],
  launchers: [
    { id: 'vscode', name: 'VS Code', command: '', args: '"{path}"', enabled: true, builtin: 'vscode' },
    { id: 'intellij', name: 'IntelliJ IDEA', command: '', args: '"{path}"', enabled: true, builtin: 'intellij' },
  ],
  defaultLauncher: 'vscode',
  presets: [],
  rules: [{ preset: 'gradle', launcher: 'intellij' }],
  projectLaunchers: {},
  markers: ['.git', 'package.json', 'README*'],
  ignore: ['node_modules'],
  filesAsProject: true,
  showEmpty: true,
  maxDepth: 20,
  followLinks: false,
  overrides: {},
  gitPath: '',
  gitInfo: true,
  gitFetch: false,
  gitFetchMinutes: 15,
  gitWarningShown: true,
  recent: [{ path: join(ROOT, 'local/awake'), launcher: 'vscode', at: Date.now() - 3600_000 }],
  startMode: 'off',
  closeToTray: false,
  hotkey: 'CmdOrCtrl+Alt+Space',
  spotlightHotkey: 'Super+Ctrl+K',
  lastPath: options.lastSelected ? parentOf(options.lastSelected) : '',
  lastSelected: options.lastSelected ?? '',
  setupDone: !options.firstRun,
}

const gitAvailable = () => !options.noGit
const state = () => ({ config: structuredClone(config), os: 'windows', version: '0.0.0-e2e', gitPath: gitAvailable() ? 'C:/Git/git.exe' : '', gitAvailable: gitAvailable(), configPath: '/cfg/config.json', home: '/home/u' })
const emitTree = () => Events.Emit('tree:updated', Tree_())
const emitConfig = () => Events.Emit('config:updated', structuredClone(config))
const fail = (code: string) => Promise.reject(new Error(code))
const Tree_ = () => toTree(fs, ROOT)

function nameProblem(name: string): string {
  if (!name.trim()) return 'name.empty'
  if (name.includes('/')) return 'name.separator'
  if (/[<>:"\\|?*]/.test(name)) return 'name.badChars'
  return ''
}

// ---------- bindings ----------
export const State = async () => state()
export const SaveConfig = async (c: any) => { log('SaveConfig', c); config = structuredClone(c); emitConfig(); emitTree(); return state() }
export const Tree = async () => Tree_()
export const Rescan = async () => { log('Rescan'); return Tree_() }
export const Dirty = async () => ({ [join(ROOT, 'github/jacopofilonzi/NtfyJS')]: 7 })
export const RefreshDirty = async () => {}
export const LauncherStatus = async () => Object.fromEntries(config.launchers.map((l: any) => [l.id, l.command || installed[l.builtin] || '']))
export const Presets = async () => catalog.map((p) => ({ ...p, builtin: true }))
export async function SyncEditors() {
  const have = new Set(config.launchers.map((l: any) => l.builtin))
  const add = Object.keys(knownNames).filter((id) => !have.has(id) && installed[id])
  if (!add.length) return []
  for (const id of add) config.launchers.push({ id, name: knownNames[id], command: '', args: '"{path}"', enabled: false, builtin: id })
  emitConfig()
  return add.map((id) => knownNames[id])
}
export const Languages = async (path: string) => ({
  stats: find(path)?.node.lang === 'Rust'
    ? [{ name: 'Rust', color: '#dea584', bytes: 9000, percent: 100 }]
    : [{ name: 'TypeScript', color: '#3178c6', bytes: 7000, percent: 70 }, { name: 'Svelte', color: '#ff3e00', bytes: 2950, percent: 29.5 }, { name: 'Other', color: '#9a9aa0', bytes: 50, percent: 0.5 }],
  partial: false,
})
export const ExportConfig = async (title: string) => { log('ExportConfig', title); return '/home/u/project-library-config.json' }
export const ImportConfig = async (title: string) => { log('ImportConfig', title); return state() }
export async function ResetConfig() {
  log('ResetConfig')
  config = { ...config, roots: [ROOT], rules: [], presets: [], projectLaunchers: {}, overrides: {}, recent: [], lastPath: '', lastSelected: '', setupDone: false }
  emitConfig()
  return state()
}

export async function Readme(path: string) {
  const f = find(path)?.node
  if (!f?.files?.includes('README.md')) return { found: false, name: '', format: 'markdown', content: '', truncated: false }
  return { found: true, name: 'README.md', format: 'markdown', content: `# ${f.name}\n\n${f.desc ?? ''}\n\n## Usage\n\nRun \`make dev\`.`, truncated: false }
}
export const GitInfo = async (path: string) => (find(path)?.node.hasGit ? structuredClone(gitOf(path)) : { isRepo: false })
export const GitChanges = async (path: string) => Array.from({ length: gitOf(path).dirty }, (_, i) => ({ status: i === 0 ? 'M' : '??', path: `src/file${i}.ts` }))
export const GitFetch = async (path: string) => { log('GitFetch', path) }
export const GitPull = async (path: string) => { log('GitPull', path); return 'Already up to date.' }

export async function Open(path: string, launcher: string) {
  log('Open', path, launcher)
  config.recent = [{ path, launcher, at: Date.now() }, ...config.recent.filter((r: any) => r.path !== path)]
  emitConfig()
}
export const TestLauncher = async (l: any, path: string) => { log('TestLauncher', l.id, path) }
export const DetectEditor = async (id: string) => installed[id] ?? ''
export const ResolveCommand = async (cmd: string) => cmd
export const Reveal = async (path: string) => { log('Reveal', path) }
export const OpenURL = async (url: string) => { log('OpenURL', url) }

export async function CheckName(parent: string, name: string) {
  const p = nameProblem(name)
  if (p) return p
  return find(parent)?.node.children?.some((c) => c.name.toLowerCase() === name.trim().toLowerCase()) ? 'exists' : ''
}
export async function Mkdir(parent: string, name: string) {
  const code = await CheckName(parent, name)
  if (code) return fail(code)
  const p = find(parent)!.node
  if (p.kind === 'empty') { p.kind = 'dir'; p.children = [] }
  p.children!.push({ name: name.trim(), kind: 'empty' })
  emitTree()
  return join(parent, name.trim())
}
export async function Rename(path: string, newName: string) {
  const f = find(path)
  if (!f?.parent) return fail('outsideRoots')
  const code = await CheckName(parentOf(path), newName)
  if (code && code !== 'exists') return fail(code)
  f.node.name = newName.trim()
  emitTree()
  return join(parentOf(path), newName.trim())
}
export const IsEmptyDir = async (path: string) => find(path)?.node.kind === 'empty'
export async function Trash(path: string) {
  log('Trash', path)
  const f = find(path)
  if (!f?.parent) return fail('outsideRoots')
  f.parent.children = f.parent.children!.filter((c) => c !== f.node)
  emitTree()
}
export async function SetOverride(path: string, kind: string) {
  if (kind) config.overrides[path] = kind
  else delete config.overrides[path]
  const n = find(path)?.node
  if (n && kind === 'project') n.kind = 'project'
  emitConfig()
  emitTree()
  return state()
}
export const CreateRoot = async (path: string) => { log('CreateRoot', path) }
export const Exists = async (path: string) => !(options.missingRoot && path === ROOT) && !!find(path)
export async function ParseRepoURL(url: string) {
  const m = url.trim().match(/github\.com[/:]([^/]+)\/([^/]+?)(?:\.git)?\/?$/)
  return m ? { ok: true, host: 'github.com', owner: m[1], repo: m[2] } : { ok: false, host: '', owner: '', repo: '' }
}
export async function Clone(id: string, url: string, parent: string, name: string) {
  log('Clone', url, parent, name)
  Events.Emit('clone:progress', { id, phase: 'Receiving objects', percent: 100 })
  let node = fs
  for (const seg of parent.slice(ROOT.length + 1).split('/').filter(Boolean)) {
    let next = node.children?.find((c) => c.name === seg)
    if (!next) { next = dir(seg, []); node.children!.push(next) }
    if (next.kind === 'empty') { next.kind = 'dir'; next.children = [] }
    node = next
  }
  node.children!.push(proj(name, 'Node', ['README.md', '.git'], { hasGit: true }))
  emitTree()
  return join(parent, name)
}
export const CancelClone = async (id: string) => { log('CancelClone', id) }
export const DetectGit = async (path: string) => (gitAvailable() || path ? 'C:/Git/git.exe' : '')
export const GitInstallInfo = async () => ({ command: 'winget install --id Git.Git -e --source winget', canRun: true, url: 'https://git-scm.com/download/win' })
export const RunGitInstall = async () => { log('RunGitInstall') }
export const PickFolder = async () => options.pickFolder ?? ''

// ---------- fake GitHub and GitLab: gh installed (depending on options.forges), glab never ----------
const ghAccount = { host: 'github.com', user: 'jacopofilonzi', protocol: 'ssh' }
export async function Forges(refresh: boolean) {
  log('Forges', refresh)
  const mode = options.forges ?? 'none'
  const gh = mode === 'none'
    ? { kind: 'github', path: '', accounts: [], message: '' }
    : { kind: 'github', path: 'C:/Program Files/GitHub CLI/gh.exe', accounts: mode === 'loggedIn' ? [ghAccount] : [], message: mode === 'loggedIn' ? '' : 'You are not logged into any GitHub hosts. To log in, run: gh auth login' }
  return [gh, { kind: 'gitlab', path: '', accounts: [], message: '' }]
}
export const ForgeInstallInfo = async (kind: string) => ({ command: `winget install --id ${kind === 'gitlab' ? 'GLab.GLab' : 'GitHub.cli'} -e --source winget`, canRun: true, url: kind === 'gitlab' ? 'https://gitlab.com/gitlab-org/cli#installation' : 'https://cli.github.com' })
export const RunForgeInstall = async (kind: string) => { log('RunForgeInstall', kind) }
const ghRepo = (fullName: string, description: string, priv = false) => ({ kind: 'github', host: 'github.com', fullName, description, private: priv, cloneUrl: `git@github.com:${fullName}.git`, web: `https://github.com/${fullName}`, updated: '2026-10-01T10:00:00Z' })
export async function ForgeRepos(refresh: boolean) {
  log('ForgeRepos', refresh)
  if (options.forges !== 'loggedIn') return { repos: [], errors: [] }
  return { repos: [ghRepo('jacopofilonzi/ShellyPlot', 'Charts for Shelly power meters'), ghRepo('dity-dev/discord-bot-java', 'Discord bot of the Dity community', true), ghRepo('jacopofilonzi/NtfyJS', 'An ntfy client for Javascript and Typescript')], errors: [] }
}
export async function ForgeInfo(remote: string, branch: string) {
  log('ForgeInfo', remote, branch)
  if (options.forges !== 'loggedIn' || !remote.includes('github.com')) return null
  const web = 'https://' + remote.replace(/^git@|\.git$/g, '').replace(':', '/')
  return {
    kind: 'github', host: 'github.com', web, prCount: 2, prMore: false, issues: 3,
    prs: [{ number: 12, title: 'Retry on 429 responses', url: web + '/pull/12', author: 'octocat', draft: false }, { number: 11, title: 'Typed events', url: web + '/pull/11', author: 'jacopofilonzi', draft: true }],
    ci: { state: 'failure', name: 'CI', url: web + '/actions/runs/1', runs: 2 },
  }
}
export const ForgeOwners = async (kind: string, host: string) => { log('ForgeOwners', kind, host); return [{ name: 'jacopofilonzi', personal: true, id: 0 }, { name: 'dity-dev', personal: false, id: 0 }] }
export async function ForgeNameTaken(kind: string, host: string, owner: string, name: string) {
  log('ForgeNameTaken', kind, host, owner, name)
  return ['jacopofilonzi/shellyplot', 'jacopofilonzi/ntfyjs', 'dity-dev/discord-bot-java'].includes(`${owner}/${name}`.toLowerCase())
}
export async function Publish(req: any) {
  log('Publish', req)
  const full = `${req.owner.name}/${req.name}`
  gitData[req.path] = { ...gitOf(req.path), remote: `git@github.com:${full}.git`, remoteWeb: `https://github.com/${full}`, hasUpstream: true }
  return { web: `https://github.com/${full}`, pushed: true, pushError: '' }
}
export const PickFile = async () => ''
export async function InitProject(path: string, git: boolean, readme: boolean) {
  log('InitProject', path, git, readme)
  const n = find(path)!.node
  n.kind = 'project'
  n.files = [...(readme ? ['README.md'] : []), ...(git ? ['.git'] : [])]
  n.hasGit = git
  emitTree()
}
export async function SetLastLocation(dir: string, selected: string) {
  config.lastPath = dir
  config.lastSelected = selected
}
export const ShowInMain = async (path: string) => { log('ShowInMain', path) }
export const RunInMain = async (cmd: string) => { log('RunInMain', cmd) }
export const OpenSpotlight = async () => { log('OpenSpotlight') }
export const HideSpotlight = async () => { log('HideSpotlight') }
export const Quit = async () => { log('Quit') }
