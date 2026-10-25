import { common, createLowlight } from 'lowlight'
import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import c from 'highlight.js/lib/languages/c'
import csharp from 'highlight.js/lib/languages/csharp'
import cpp from 'highlight.js/lib/languages/cpp'
import css from 'highlight.js/lib/languages/css'
import dockerfile from 'highlight.js/lib/languages/dockerfile'
import go from 'highlight.js/lib/languages/go'
import graphql from 'highlight.js/lib/languages/graphql'
import java from 'highlight.js/lib/languages/java'
import javascript from 'highlight.js/lib/languages/javascript'
import json from 'highlight.js/lib/languages/json'
import kotlin from 'highlight.js/lib/languages/kotlin'
import markdown from 'highlight.js/lib/languages/markdown'
import php from 'highlight.js/lib/languages/php'
import python from 'highlight.js/lib/languages/python'
import ruby from 'highlight.js/lib/languages/ruby'
import rust from 'highlight.js/lib/languages/rust'
import scss from 'highlight.js/lib/languages/scss'
import sql from 'highlight.js/lib/languages/sql'
import swift from 'highlight.js/lib/languages/swift'
import typescript from 'highlight.js/lib/languages/typescript'
import xml from 'highlight.js/lib/languages/xml'
import yaml from 'highlight.js/lib/languages/yaml'

/** TipTap / lowlight instance (common pack covers most ids below). */
export const codeLowlight = createLowlight(common)

const HLJS_LANGS: [string, typeof javascript][] = [
  ['bash', bash],
  ['c', c],
  ['csharp', csharp],
  ['cpp', cpp],
  ['css', css],
  ['dockerfile', dockerfile],
  ['go', go],
  ['graphql', graphql],
  ['java', java],
  ['javascript', javascript],
  ['json', json],
  ['kotlin', kotlin],
  ['markdown', markdown],
  ['php', php],
  ['python', python],
  ['ruby', ruby],
  ['rust', rust],
  ['scss', scss],
  ['sql', sql],
  ['swift', swift],
  ['typescript', typescript],
  ['xml', xml],
  ['yaml', yaml],
]

for (const [name, def] of HLJS_LANGS) {
  hljs.registerLanguage(name, def)
}

/** Aliases → canonical highlight.js / lowlight id. */
const LANG_ALIASES: Record<string, string> = {
  js: 'javascript',
  jsx: 'javascript',
  mjs: 'javascript',
  cjs: 'javascript',
  ts: 'typescript',
  tsx: 'typescript',
  py: 'python',
  rb: 'ruby',
  rs: 'rust',
  sh: 'bash',
  shell: 'bash',
  zsh: 'bash',
  yml: 'yaml',
  md: 'markdown',
  htm: 'xml',
  html: 'xml',
  svg: 'xml',
  cs: 'csharp',
  'c++': 'cpp',
  'c#': 'csharp',
  kt: 'kotlin',
  plaintext: '',
  text: '',
  plain: '',
}

export type CodeLanguageOption = {
  id: string
  label: string
}

/** Dropdown options for the editor language picker. */
export const CODE_LANGUAGE_OPTIONS: CodeLanguageOption[] = [
  { id: '', label: 'Plain text' },
  { id: 'javascript', label: 'JavaScript' },
  { id: 'typescript', label: 'TypeScript' },
  { id: 'go', label: 'Go' },
  { id: 'python', label: 'Python' },
  { id: 'java', label: 'Java' },
  { id: 'rust', label: 'Rust' },
  { id: 'c', label: 'C' },
  { id: 'cpp', label: 'C++' },
  { id: 'csharp', label: 'C#' },
  { id: 'php', label: 'PHP' },
  { id: 'ruby', label: 'Ruby' },
  { id: 'swift', label: 'Swift' },
  { id: 'kotlin', label: 'Kotlin' },
  { id: 'sql', label: 'SQL' },
  { id: 'json', label: 'JSON' },
  { id: 'yaml', label: 'YAML' },
  { id: 'xml', label: 'HTML / XML' },
  { id: 'css', label: 'CSS' },
  { id: 'scss', label: 'SCSS' },
  { id: 'bash', label: 'Bash' },
  { id: 'markdown', label: 'Markdown' },
  { id: 'dockerfile', label: 'Dockerfile' },
  { id: 'graphql', label: 'GraphQL' },
]

export function normalizeCodeLanguage(raw: string | null | undefined): string {
  const key = String(raw ?? '')
    .trim()
    .toLowerCase()
  if (!key) return ''
  if (key in LANG_ALIASES) return LANG_ALIASES[key]
  return key
}

function escapeHtml(s: string) {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

/** Reader-side highlight → HTML (for dangerouslySetInnerHTML). */
export function highlightCodeToHtml(code: string, language?: string | null): string {
  const lang = normalizeCodeLanguage(language)
  if (!lang || !hljs.getLanguage(lang)) {
    return escapeHtml(code)
  }
  try {
    return hljs.highlight(code, { language: lang, ignoreIllegals: true }).value
  } catch {
    return escapeHtml(code)
  }
}

export function codeLanguageLabel(language?: string | null): string {
  const lang = normalizeCodeLanguage(language)
  if (!lang) return ''
  const opt = CODE_LANGUAGE_OPTIONS.find((o) => o.id === lang)
  return opt?.label ?? lang
}
