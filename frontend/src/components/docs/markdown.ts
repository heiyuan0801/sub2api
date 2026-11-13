import { Marked } from 'marked'
import DOMPurify from 'dompurify'
import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import python from 'highlight.js/lib/languages/python'
import json from 'highlight.js/lib/languages/json'
import yaml from 'highlight.js/lib/languages/yaml'
import go from 'highlight.js/lib/languages/go'
import sql from 'highlight.js/lib/languages/sql'

for (const [name, language] of Object.entries({ bash, javascript, typescript, python, json, yaml, go, sql })) {
  hljs.registerLanguage(name, language)
}
hljs.registerAliases(['sh', 'shell', 'zsh'], { languageName: 'bash' })
hljs.registerAliases(['js', 'jsx'], { languageName: 'javascript' })
hljs.registerAliases(['ts', 'tsx'], { languageName: 'typescript' })
hljs.registerAliases(['py'], { languageName: 'python' })

const markdownParser = new Marked({ gfm: true, async: false, renderer: { html: () => '' } })
export interface DocHeading { id: string; text: string; level: number }
export const docImagePath = /^\/api\/v1\/docs\/images\/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/i

export function renderDocMarkdown(markdown: string, copyLabel: string, admonitions: Record<string, string>, anchorPrefix = '') {
  const template = document.createElement('template')
  template.innerHTML = DOMPurify.sanitize(markdownParser.parse(markdown, { async: false }), {
    USE_PROFILES: { html: true },
    FORBID_TAGS: ['form', 'input', 'style', 'iframe', 'button'],
    FORBID_ATTR: ['style', 'srcset', 'id', 'name', 'target'],
    ALLOW_DATA_ATTR: false,
  })
  const headings: DocHeading[] = []
  const ids = new Set<string>()
  for (const heading of template.content.querySelectorAll('h1,h2,h3,h4,h5,h6')) {
    const text = heading.textContent?.trim() || ''
    const base = (text.toLowerCase().replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-+|-+$/g, '') || 'section')
    let id = base
    for (let suffix = 2; ids.has(id); suffix++) id = `${base}-${suffix}`
    ids.add(id)
    heading.id = anchorPrefix + id
    const level = Number(heading.tagName.slice(1))
    if (level <= 4) headings.push({ id, text, level })
  }
  for (const quote of template.content.querySelectorAll('blockquote')) {
    const first = quote.firstElementChild
    const initial = first?.firstChild
    const match = initial?.textContent?.match(/^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\](?:\s|$)/)
    if (initial?.nodeType !== Node.TEXT_NODE || !match) continue
    initial.textContent = (initial.textContent || '').slice(match[0].length).trimStart()
    const label = document.createElement('strong')
    label.className = 'docs-callout-label'
    label.textContent = admonitions[match[1]] || match[1]
    quote.prepend(label)
    quote.classList.add('docs-callout', `docs-callout-${match[1].toLowerCase()}`)
  }
  for (const code of template.content.querySelectorAll('pre > code')) {
    const pre = code.parentElement!
    const language = [...code.classList].find(c => c.startsWith('language-'))?.slice(9).toLowerCase()
    if (language && hljs.getLanguage(language) && (code.textContent?.length || 0) <= 32000) {
      try {
        code.innerHTML = hljs.highlight(code.textContent || '', { language, ignoreIllegals: true }).value
      } catch {
        // The original escaped text remains usable even if highlighting fails.
      }
    }
    const block = document.createElement('div')
    block.className = 'docs-code'
    const header = document.createElement('div')
    header.className = 'docs-code-header'
    const title = document.createElement('span')
    title.textContent = language || 'text'
    const copy = document.createElement('button')
    copy.type = 'button'
    copy.className = 'docs-copy'
    copy.textContent = copyLabel
    header.append(title, copy)
    pre.replaceWith(block)
    block.append(header, pre)
  }
  for (const table of template.content.querySelectorAll('table')) {
    const scroll = document.createElement('div')
    scroll.className = 'docs-table'
    scroll.tabIndex = 0
    table.replaceWith(scroll)
    scroll.append(table)
  }
  for (const img of template.content.querySelectorAll('img')) {
    const source = img.getAttribute('src') || ''
    if (docImagePath.test(source)) {
      img.dataset.docsImage = source.match(docImagePath)![1]
      img.removeAttribute('src')
    } else if (!/^https:\/\//i.test(source)) {
      img.removeAttribute('src')
    }
    img.loading = 'lazy'
    img.decoding = 'async'
    img.referrerPolicy = 'no-referrer'
  }
  for (const link of template.content.querySelectorAll<HTMLAnchorElement>('a[href]')) {
    const href = link.getAttribute('href') || ''
    if (/^https?:\/\//i.test(href)) {
      link.target = '_blank'
      link.rel = 'noopener noreferrer'
    }
  }
  return { html: template.innerHTML, headings }
}
