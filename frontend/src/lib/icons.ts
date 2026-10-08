// Inline SVG icons (stroke, color inherited from currentColor).
const svg = (size: number, body: string, extra = '') =>
  `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" ${extra}>${body}</svg>`

export const icons = {
  folder: (s = 16) => svg(s, '<path d="M3 6h6l2 2h10v11H3z"/>', 'class="ico"'),
  search: (s = 15) => svg(s, '<circle cx="11" cy="11" r="7"/><path d="M20 20l-3.5-3.5"/>'),
  clone: (s = 16) => svg(s, '<path d="M12 4v12M6 10l6 6 6-6M4 20h16"/>'),
  sun: (s = 18) => svg(s, '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>'),
  moon: (s = 18) => svg(s, '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>'),
  gear: (s = 18) =>
    svg(s, '<path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>'),
  branch: (s = 13) => svg(s, '<circle cx="6" cy="5" r="2"/><circle cx="6" cy="19" r="2"/><circle cx="18" cy="8" r="2"/><path d="M6 7v10M18 10c0 4-6 3-12 7"/>'),
}

/** colors per language (dot in the column, icon in the palette) */
export const langColor: Record<string, string> = {
  Go: '#29beb0', Rust: '#e3a27a', Node: '#6cc24a', Deno: '#3d3d3d', Java: '#e76f00', Python: '#3572a5',
  PHP: '#7a86b8', Ruby: '#cc342d', 'C#': '#7b5ea7', Dart: '#00b4ab', Elixir: '#6e4a7e', 'C/C++': '#5c6bc0',
}
export const defaultLangColor = '#a78bfa'
export const colorOf = (lang?: string) => (lang && langColor[lang]) || defaultLangColor

export const initials = (name: string) => name.replace(/[^a-z0-9]/gi, '').slice(0, 2).toUpperCase() || '?'
