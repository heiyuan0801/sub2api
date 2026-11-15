import { readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, extname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import postcss from 'postcss'
import ts from 'typescript'
import { parse } from 'vue/compiler-sfc'

// Exact pairs only. The same non-theme variants must occur on both sides.
// Keep explicit palette choices that lack a matching semantic token.
const themePairs = [
  ['bg-white', 'bg-dark-800', 'bg-card'],
  ['bg-white', 'bg-dark-900', 'bg-background'],
  ['bg-gray-50', 'bg-dark-950', 'bg-canvas'],
  ['bg-gray-100', 'bg-dark-800', 'bg-muted'],
  ['text-gray-900', 'text-white', 'text-foreground'],
  ['text-gray-900', 'text-gray-100', 'text-foreground'],
  ['text-gray-500', 'text-gray-400', 'text-muted-foreground'],
  ['text-gray-500', 'text-dark-400', 'text-muted-foreground'],
  ['border-gray-200', 'border-dark-700', 'border-border'],
  ['border-gray-200', 'border-dark-600', 'border-border'],
  ['divide-gray-200', 'divide-dark-700', 'divide-border'],
  ['text-primary-600', 'text-primary-400', 'text-brand'],
  ['text-primary-700', 'text-primary-300', 'text-brand'],
  ['text-primary-500', 'text-primary-300', 'text-brand'],
  ['bg-primary-50', 'bg-primary-900/20', 'bg-brand/10'],
  ['bg-primary-50', 'bg-primary-900/30', 'bg-brand/10'],
  ['bg-primary-100', 'bg-primary-900/30', 'bg-brand/15'],
  ['bg-primary-50', 'bg-primary-500/10', 'bg-brand/10'],
]

// Mid-range primary utilities used for actions and focus/selection accents.
// Preserve alpha modifiers. Leave gradients and light/dark-only shades alone.
const brandUtilities = /^(bg|text|border(?:-[trblxyse])?|ring|outline|accent|decoration|shadow)-primary-(500|600|700)(\/\d+)?$/
const colorUtility = /^(bg|text|border(?:-[trblxyse])?|ring|outline|accent|decoration|shadow|divide)-(?:\d|\[|[a-z])/
const primaryScale = /\bprimary-(?:50|[1-9]00|950)(?![\w-])/g
const extensions = new Set(['.vue', '.css', '.ts', '.js', '.tsx', '.jsx'])

function parseUtility(token, index) {
  const parts = token.split(':')
  const utility = parts.pop()
  const dark = parts.includes('dark')
  const prefix = parts.filter(part => part !== 'dark').join(':')
  return { index, utility, dark, prefix, kind: utility.match(colorUtility)?.[1] }
}

function migrateClasses(source, partialStart = false, partialEnd = false) {
  const matches = [...source.matchAll(/\S+/g)]
  const tokens = matches
    .map((match, index) => {
      if ((partialStart && match.index === 0) || (partialEnd && match.index + match[0].length === source.length)) return null
      // Leave arbitrary selectors, !important and escaped classes verbatim.
      if (!/^[\w:/.-]+$/.test(match[0])) return null
      return parseUtility(match[0], index)
    })
    .filter(Boolean)
  const edits = new Map()
  const add = (token, value) => edits.set(token.index, token.prefix ? token.prefix + ':' + value : value)

  for (const token of tokens) {
    if (token.dark || edits.has(token.index)) continue
    const darkAlternatives = tokens.filter(other => other.dark && other.prefix === token.prefix && other.kind === token.kind)
    if (darkAlternatives.length === 1) {
      const dark = darkAlternatives[0]
      const pair = themePairs.find(([lightName, darkName]) => lightName === token.utility && darkName === dark.utility)
      const sameLight = tokens.filter(other => !other.dark && other.prefix === token.prefix && other.utility === token.utility)
      if (pair && sameLight.length === 1 && !edits.has(dark.index)) {
        add(token, pair[2])
        edits.set(dark.index, null)
      }
      continue
    }
    if (darkAlternatives.length > 0) continue
    const brand = token.utility.match(brandUtilities)
    if (!brand) continue
    const [, kind, , alpha] = brand
    const hoverAlpha = kind === 'bg' && /(?:^|:)hover$/.test(token.prefix) && !alpha ? '/90' : ''
    add(token, kind + '-brand' + (alpha || hoverAlpha))

    // Keep foreground contrast when an explicit white label belongs to a solid action.
    if (kind === 'bg' && !alpha && !token.prefix && !tokens.some(other => other.dark && other.utility.startsWith('text-'))) {
      const white = tokens.find(other => !other.dark && !other.prefix && other.utility === 'text-white')
      if (white) add(white, 'text-brand-foreground')
    }
  }

  // Remove the consumed class and the whitespace before it, keeping attribute
  // line breaks stable for all unaffected classes.
  let result = source
  for (const [index, replacement] of [...edits].sort((a, b) => b[0] - a[0])) {
    const match = matches[index]
    let start = match.index
    const end = start + match[0].length
    if (replacement === null) {
      while (start > 0 && /[ \t]/.test(source[start - 1])) start--
    }
    result = result.slice(0, start) + (replacement ?? '') + result.slice(end)
  }
  return result
}

function migrateSource(source, extension, filename) {
  const edits = []
  function edit(start, length, partialStart, partialEnd) {
    const original = source.slice(start, start + length)
    const updated = migrateClasses(original, partialStart, partialEnd)
    if (original !== updated) edits.push({ start, end: start + length, updated })
  }
  function script(content, offset, scriptFilename = filename) {
    const ast = ts.createSourceFile(scriptFilename, content, ts.ScriptTarget.Latest, true)
    function visit(node) {
      if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
        // Ignore compound text and CSS/DOM selectors. A plain class list may
        // include an arbitrary Tailwind utility whose internals stay untouched.
        if (/^[\s!\w[&:+./%#=\]'"-]+$/.test(node.text) &&
            /(?:^|\s)(?:[\w-]+:)*(?:bg|border|text|ring|outline|accent|shadow|divide|decoration)-[\w-]+/.test(node.text)) {
          edit(offset + node.getStart(ast) + 1, node.getWidth(ast) - 2)
        }
      } else if (ts.isTemplateHead(node) || ts.isTemplateMiddle(node) || ts.isTemplateTail(node)) {
        const tail = ts.isTemplateTail(node)
        edit(offset + node.getStart(ast) + 1, node.getWidth(ast) - (tail ? 2 : 3), !ts.isTemplateHead(node), !tail)
      }
      ts.forEachChild(node, visit)
    }
    visit(ast)
  }
  function css(content, offset) {
    postcss.parse(content, { from: filename }).walkAtRules('apply', rule => {
      const start = rule.source.start.offset + '@apply'.length + (rule.raws.afterName ?? '').length
      edit(offset + start, rule.params.length)
    })
  }

  if (extension === '.vue') {
    const { descriptor, errors } = parse(source, { filename })
    if (errors.length) throw new Error('Cannot parse ' + filename + ': ' + errors.join('; '))
    const visitTemplate = (node) => {
      for (const prop of node.props ?? []) {
        // Vue compiler AST nodes 6 and 7 are HTML attributes and directives.
        if (prop.type === 6 && prop.name === 'class' && prop.value) {
          const { loc } = prop.value
          const quoted = /^[\x22']/.test(loc.source)
          edit(loc.start.offset + Number(quoted), loc.source.length - (quoted ? 2 : 0))
        }
        if (prop.type === 7 && prop.name === 'bind' && prop.arg?.isStatic && prop.arg.content === 'class' && prop.exp) {
          script('(' + prop.exp.loc.source + ')', prop.exp.loc.start.offset - 1, 'class.ts')
        }
      }
      for (const child of node.children ?? []) visitTemplate(child)
    }
    if (descriptor.template?.ast) visitTemplate(descriptor.template.ast)
    for (const block of [descriptor.script, descriptor.scriptSetup]) {
      if (block) script(block.content, block.loc.start.offset, 'script.' + (block.lang || 'js'))
    }
    for (const block of descriptor.styles) {
      if (!block.lang || block.lang === 'css') css(block.content, block.loc.start.offset)
    }
  } else if (extension === '.css') css(source, 0)
  else script(source, 0)

  let result = source
  for (const { start, end, updated } of edits.sort((a, b) => b.start - a.start)) {
    result = result.slice(0, start) + updated + result.slice(end)
  }
  return result
}

function *sourceFiles(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    const filename = resolve(directory, entry.name)
    if (entry.isDirectory() && entry.name !== '__tests__') yield *sourceFiles(filename)
    if (entry.isFile() && extensions.has(extname(filename)) && !/\.(?:spec|test|d)\.[jt]sx?$/.test(entry.name)) yield filename
  }
}

const args = process.argv.slice(2)
if (args.some(arg => arg !== '--check')) throw new Error('Usage: pnpm codemod:colors [--check]')
const dryRun = args.includes('--check')
const totals = { darkBefore: 0, darkAfter: 0, numericBefore: 0, numericAfter: 0, filesChanged: 0 }
const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../src')
for (const filename of sourceFiles(root)) {
  const before = readFileSync(filename, 'utf8')
  const after = migrateSource(before, extname(filename), filename)
  totals.darkBefore += (before.match(/\bdark:/g) ?? []).length
  totals.darkAfter += (after.match(/\bdark:/g) ?? []).length
  totals.numericBefore += (before.match(primaryScale) ?? []).length
  totals.numericAfter += (after.match(primaryScale) ?? []).length
  if (before !== after) {
    totals.filesChanged++
    if (!dryRun) writeFileSync(filename, after)
  }
}
console.log(JSON.stringify(totals, null, 2))
if (dryRun && totals.filesChanged) process.exitCode = 1
