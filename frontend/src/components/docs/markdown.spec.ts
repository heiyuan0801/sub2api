import { describe, expect, it } from 'vitest'
import { renderDocMarkdown } from './markdown'

const render = (source: string) => renderDocMarkdown(source, 'Copy', { TIP: 'Tip' }, 'doc-')
const image = '/api/v1/docs/images/0653b62e-3e74-4f6b-8692-3d463052e871'

describe('documentation Markdown', () => {
  it('builds stable Unicode headings and enhances code, tables and callouts', () => {
    const result = render('## 快速开始\n\n## 快速开始\n\n> [!TIP]\n> Copy this.\n\n```python\nprint("test")\n```\n\n| A | B |\n| - | - |\n| c | d |')
    expect(result.headings).toEqual([
      { id: '快速开始', level: 2, text: '快速开始' }, { id: '快速开始-2', level: 2, text: '快速开始' },
    ])
    expect(result.html).toContain('id="doc-快速开始"')
    expect(result.html).toContain('docs-callout-tip')
    expect(result.html).toContain('docs-table')
    expect(result.html).toContain('hljs-string')
    expect(result.html).toContain('docs-copy')
  })

  it('keeps uploaded images behind authenticated fetches and strips unsafe content', () => {
    const result = render(`![private](${image})\n\n[x](javascript:alert%281%29)\n\n<img src=x onerror=alert(1)>\n\n![foreign](//evil.example/tracker.png)\n\n[good](https://example.com)`)
    expect(result.html).toContain('data-docs-image="0653b62e-3e74-4f6b-8692-3d463052e871"')
    expect(result.html).not.toContain('src=')
    expect(result.html).not.toMatch(/javascript:|onerror|evil\.example/i)
    expect(result.html).toContain('rel="noopener noreferrer"')
  })
})
