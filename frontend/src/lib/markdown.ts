// Rendering dei README: marked per il markdown, DOMPurify perché il contenuto non è fidato.
import { Marked } from 'marked'
import DOMPurify from 'dompurify'

const isAbsoluteUrl = (u: string) => /^[a-z][a-z0-9+.-]*:/i.test(u) || u.startsWith('//')

/** percorso di un file del progetto, servito dal backend (solo immagini dentro le radici) */
function projectFileUrl(projectDir: string, rel: string): string {
  const clean = decodeURIComponent(rel.split('#')[0].split('?')[0]).replace(/^\.\//, '')
  const sep = projectDir.includes('\\') ? '\\' : '/'
  const full = projectDir + sep + clean.replace(/\//g, sep)
  return '/project-file?p=' + encodeURIComponent(full)
}

export function renderMarkdown(md: string, projectDir: string): string {
  const marked = new Marked({ gfm: true, breaks: false })
  const html = marked.parse(md, { async: false }) as string
  const clean = DOMPurify.sanitize(html, { USE_PROFILES: { html: true }, FORBID_TAGS: ['style', 'form', 'input'] })

  // immagini con percorso relativo → servite dal backend; link marcati per la gestione dei click
  const doc = new DOMParser().parseFromString(clean, 'text/html')
  doc.querySelectorAll('img').forEach((img) => {
    const src = img.getAttribute('src') ?? ''
    if (src && !isAbsoluteUrl(src) && !src.startsWith('/')) img.setAttribute('src', projectFileUrl(projectDir, src))
    img.setAttribute('loading', 'lazy')
  })
  doc.querySelectorAll('a').forEach((a) => a.setAttribute('rel', 'noreferrer'))
  return doc.body.innerHTML
}
