import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  NodeViewContent,
  NodeViewWrapper,
  type NodeViewProps,
} from '@tiptap/react'
import { Check, Copy } from 'lucide-react'
import {
  CODE_LANGUAGE_OPTIONS,
  normalizeCodeLanguage,
} from '@/lib/code-languages'
import { cn } from '@/lib/utils'

export function CodeBlockNodeView({ node, updateAttributes }: NodeViewProps) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  const language = normalizeCodeLanguage(
    (node.attrs.language as string | null | undefined) ?? '',
  )
  const title = String(node.attrs.title ?? '')

  async function copyCode() {
    const text = node.textContent ?? ''
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      // ignore clipboard failures (insecure context / denied)
    }
  }

  return (
    <NodeViewWrapper
      className="code-block-node relative my-3 overflow-hidden rounded-lg border border-border bg-surface-muted"
      data-language={language || undefined}
      data-title={title.trim() || undefined}
    >
      <div
        className="flex items-center justify-between gap-2 border-b border-border px-3 py-1.5"
        contentEditable={false}
      >
        <input
          type="text"
          value={title}
          placeholder={t('editor.toolbar.codeTitlePlaceholder')}
          aria-label={t('editor.toolbar.codeTitle')}
          className="min-w-0 flex-1 truncate border-0 bg-transparent px-0.5 font-mono text-[11px] font-medium text-zinc-800 outline-none placeholder:text-zinc-400"
          onChange={(e) => updateAttributes({ title: e.target.value })}
          onMouseDown={(e) => e.stopPropagation()}
          onKeyDown={(e) => e.stopPropagation()}
        />
        <div className="flex shrink-0 items-center gap-1.5">
          <select
            aria-label={t('editor.toolbar.codeLanguage')}
            className="h-7 max-w-[10rem] rounded-md border border-border bg-white px-1.5 font-mono text-[11px] text-zinc-800 outline-none focus:border-zinc-400"
            value={language}
            onChange={(e) => {
              const next = normalizeCodeLanguage(e.target.value) || null
              updateAttributes({ language: next })
            }}
            onMouseDown={(e) => e.stopPropagation()}
          >
            {CODE_LANGUAGE_OPTIONS.map((opt) => (
              <option key={opt.id || 'plain'} value={opt.id}>
                {opt.id ? opt.label : t('editor.toolbar.codeLanguagePlain')}
              </option>
            ))}
          </select>
          <button
            type="button"
            aria-label={copied ? t('editor.toolbar.codeCopied') : t('editor.toolbar.codeCopy')}
            title={copied ? t('editor.toolbar.codeCopied') : t('editor.toolbar.codeCopy')}
            className={cn(
              'inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-border bg-white text-zinc-600 transition-colors hover:bg-zinc-50 hover:text-zinc-900',
              copied && 'border-emerald-200 text-emerald-600',
            )}
            onClick={() => void copyCode()}
            onMouseDown={(e) => e.stopPropagation()}
          >
            {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
          </button>
        </div>
      </div>
      <pre className="m-0 overflow-x-auto bg-transparent p-4 font-mono text-[13px] leading-6">
        <NodeViewContent as="code" className="hljs" />
      </pre>
    </NodeViewWrapper>
  )
}
