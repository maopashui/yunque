import { useMemo, useState, type CSSProperties, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Copy } from 'lucide-react'
import type { Block } from '@/lib/api'
import {
  codeLanguageLabel,
  highlightCodeToHtml,
} from '@/lib/code-languages'
import { safeImageSrc, safeLinkHref } from '@/lib/safe-url'
import { cn } from '@/lib/utils'

type Inline = {
  type: string
  text?: string
  marks?: { type: string; attrs?: Record<string, unknown> }[]
  attrs?: Record<string, unknown>
}

const CODE_COLLAPSE_LINES = 24
const TABLE_COLLAPSE_ROWS = 12

function renderInlines(inlines: Inline[] | undefined) {
  if (!inlines?.length) return null
  return inlines.map((node, i) => {
    if (node.type === 'image') {
      const src = safeImageSrc(node.attrs?.src)
      const alt = String(node.attrs?.alt ?? '')
      if (!src) return null
      return (
        <img
          key={i}
          src={src}
          alt={alt}
          className="my-2 max-w-full rounded-md border border-border"
        />
      )
    }
    const raw = node.text ?? ''
    // Hard breaks are stored as "\n" — render as <br> so reading view matches the editor.
    const withBreaks: ReactNode = raw.includes('\n')
      ? raw.split('\n').map((part, j, parts) => (
          <span key={`${i}-${j}`}>
            {part}
            {j < parts.length - 1 ? <br /> : null}
          </span>
        ))
      : raw

    let el: ReactNode = withBreaks
    const style: CSSProperties = {}
    for (const mark of node.marks ?? []) {
      switch (mark.type) {
        case 'bold':
          el = <strong key={`b-${i}`}>{el}</strong>
          break
        case 'italic':
          el = <em key={`i-${i}`}>{el}</em>
          break
        case 'strike':
          el = <s key={`s-${i}`}>{el}</s>
          break
        case 'underline':
          el = <u key={`u-${i}`}>{el}</u>
          break
        case 'code':
          el = (
            <code
              key={`c-${i}`}
              className="rounded bg-surface-hover px-1 font-mono text-[13px]"
            >
              {el}
            </code>
          )
          break
        case 'link': {
          const href = safeLinkHref(mark.attrs?.href)
          if (href) {
            el = (
              <a
                key={`a-${i}`}
                href={href}
                className="underline underline-offset-2"
                target="_blank"
                rel="noreferrer"
              >
                {el}
              </a>
            )
          }
          break
        }
        case 'textStyle': {
          const color = mark.attrs?.color
          if (typeof color === 'string' && color) style.color = color
          break
        }
        case 'highlight': {
          const bg = mark.attrs?.color
          if (typeof bg === 'string' && bg) style.backgroundColor = bg
          else style.backgroundColor = '#fef08a'
          break
        }
      }
    }
    return (
      <span key={i} style={Object.keys(style).length ? style : undefined}>
        {el}
      </span>
    )
  })
}

function listItems(items: unknown, ordered: boolean, depth = 0): ReactNode {
  const list = (items as { inlines?: Inline[]; children?: unknown[] }[]) ?? []
  const Tag = ordered ? 'ol' : 'ul'
  return (
    <Tag
      className={cn(
        'my-2 list-inside',
        ordered ? 'list-decimal' : 'list-disc',
        depth > 0 && 'ml-4',
      )}
    >
      {list.map((item, i) => (
        <li key={i} className="my-1">
          {renderInlines(item.inlines)}
          {item.children ? listItems(item.children, ordered, depth + 1) : null}
        </li>
      ))}
    </Tag>
  )
}

function ExpandControl({
  expanded,
  onToggle,
  collapsedLabel,
  expandedLabel,
}: {
  expanded: boolean
  onToggle: () => void
  collapsedLabel: string
  expandedLabel: string
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className="font-mono text-[11px] text-muted underline-offset-2 hover:text-foreground hover:underline"
    >
      {expanded ? expandedLabel : collapsedLabel}
    </button>
  )
}

function CodeBlockView({
  text,
  fullText,
  title,
  language,
  langLabel,
  long,
  expanded,
  onToggle,
  expandLabel,
  collapseLabel,
}: {
  text: string
  fullText: string
  title: string
  language: string
  langLabel: string
  long: boolean
  expanded: boolean
  onToggle: () => void
  expandLabel: string
  collapseLabel: string
}) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  const html = useMemo(
    () => highlightCodeToHtml(text, language),
    [text, language],
  )

  async function copyCode() {
    try {
      await navigator.clipboard.writeText(fullText)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      // ignore
    }
  }

  return (
    <div className="my-3 overflow-hidden rounded-lg border border-border bg-surface-muted">
      <div className="flex items-center justify-between gap-2 border-b border-border px-3 py-1.5">
        <span
          className={cn(
            'min-w-0 flex-1 truncate px-0.5 font-mono text-[11px] font-medium',
            title ? 'text-zinc-800' : 'text-zinc-400',
          )}
        >
          {title || t('editor.toolbar.codeTitlePlaceholder')}
        </span>
        <div className="flex shrink-0 items-center gap-1.5">
          <span className="inline-flex h-7 max-w-[10rem] items-center truncate rounded-md border border-border bg-white px-2 font-mono text-[11px] text-zinc-800">
            {langLabel || t('editor.toolbar.codeLanguagePlain')}
          </span>
          <button
            type="button"
            aria-label={copied ? t('editor.toolbar.codeCopied') : t('editor.toolbar.codeCopy')}
            title={copied ? t('editor.toolbar.codeCopied') : t('editor.toolbar.codeCopy')}
            className={cn(
              'inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-border bg-white text-zinc-600 transition-colors hover:bg-zinc-50 hover:text-zinc-900',
              copied && 'border-emerald-200 text-emerald-600',
            )}
            onClick={() => void copyCode()}
          >
            {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
          </button>
        </div>
      </div>
      <pre
        className={cn(
          'm-0 overflow-x-auto bg-transparent p-4 font-mono text-[13px] leading-6',
          long && !expanded && 'max-h-72 overflow-y-hidden',
          long && expanded && 'max-h-[480px] overflow-y-auto',
        )}
      >
        <code className="hljs" dangerouslySetInnerHTML={{ __html: html }} />
      </pre>
      {long ? (
        <div className="border-t border-border px-3 py-1.5">
          <ExpandControl
            expanded={expanded}
            onToggle={onToggle}
            collapsedLabel={expandLabel}
            expandedLabel={collapseLabel}
          />
        </div>
      ) : null}
    </div>
  )
}

export function BlockView({ block }: { block: Block }) {
  const { t } = useTranslation()
  const content = block.content ?? {}
  const attrs = block.attrs ?? {}
  const [expanded, setExpanded] = useState(false)

  switch (block.block_type) {
    case 'heading': {
      const level = Number(attrs.level ?? 1)
      const className = cn(
        'font-semibold tracking-tight text-foreground',
        level <= 1 && 'text-3xl mt-6 mb-3',
        level === 2 && 'text-2xl mt-5 mb-2',
        level === 3 && 'text-xl mt-4 mb-2',
        level >= 4 && 'text-lg mt-3 mb-2',
      )
      const children = renderInlines(content.inlines as Inline[])
      if (level <= 1) return <h1 className={className}>{children}</h1>
      if (level === 2) return <h2 className={className}>{children}</h2>
      if (level === 3) return <h3 className={className}>{children}</h3>
      if (level === 4) return <h4 className={className}>{children}</h4>
      if (level === 5) return <h5 className={className}>{children}</h5>
      return <h6 className={className}>{children}</h6>
    }
    case 'paragraph':
      return (
        <p className="my-3 text-[15px] leading-7">
          {renderInlines(content.inlines as Inline[])}
        </p>
      )
    case 'bullet_list':
      return listItems(content.items, false)
    case 'ordered_list':
      return listItems(content.items, true)
    case 'code_block': {
      const text = String(content.text ?? '')
      const language = String(attrs.language ?? '')
      const title = String(attrs.title ?? '').trim()
      const langLabel = codeLanguageLabel(language)
      const lines = text.split('\n')
      const long = lines.length > CODE_COLLAPSE_LINES || text.length > 4000
      const shown =
        !long || expanded ? text : lines.slice(0, CODE_COLLAPSE_LINES).join('\n') + '\n…'
      return (
        <CodeBlockView
          text={shown}
          fullText={text}
          title={title}
          language={language}
          langLabel={langLabel}
          long={long}
          expanded={expanded}
          onToggle={() => setExpanded((v) => !v)}
          expandLabel={t('blockView.expandCode', { count: lines.length })}
          collapseLabel={t('blockView.collapseCode')}
        />
      )
    }
    case 'blockquote':
      return (
        <blockquote className="my-3 border-l-2 border-foreground pl-4 text-muted-foreground">
          {Array.isArray(content.blocks)
            ? (content.blocks as Record<string, unknown>[]).map((inner, i) => (
                <BlockView
                  key={i}
                  block={{
                    id: `${block.id}-q-${i}`,
                    document_id: block.document_id,
                    order_key: String(i),
                    block_type: String(inner.block_type ?? 'paragraph'),
                    content: (inner.content as Record<string, unknown>) ?? {},
                    attrs: (inner.attrs as Record<string, unknown>) ?? {},
                    revision: 1,
                    created_at: block.created_at,
                    updated_at: block.updated_at,
                  }}
                />
              ))
            : null}
        </blockquote>
      )
    case 'thematic_break':
      return <hr className="my-6 border-border" />
    case 'image': {
      const src = safeImageSrc(content.src)
      if (!src) return null
      return (
        <figure className="my-4">
          <img
            src={src}
            alt={String(content.alt ?? '')}
            className="max-w-full rounded-md border border-border"
          />
        </figure>
      )
    }
    case 'table': {
      const rows = (content.rows as Inline[][][]) ?? []
      const long = rows.length > TABLE_COLLAPSE_ROWS
      const shown = !long || expanded ? rows : rows.slice(0, TABLE_COLLAPSE_ROWS)
      return (
        <div className="my-4">
          <div
            className={cn(
              'overflow-x-auto rounded-md border border-border',
              long && !expanded && 'max-h-72 overflow-y-hidden',
              long && expanded && 'max-h-[480px] overflow-y-auto',
            )}
          >
            <table className="w-full border-collapse text-sm">
              <tbody>
                {shown.map((row, ri) => (
                  <tr key={ri} className="border-b border-border">
                    {row.map((cell, ci) => (
                      <td key={ci} className="px-3 py-2 text-left">
                        {renderInlines(cell)}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {long ? (
            <ExpandControl
              expanded={expanded}
              onToggle={() => setExpanded((v) => !v)}
              collapsedLabel={t('blockView.expandTable', { count: rows.length })}
              expandedLabel={t('blockView.collapseTable')}
            />
          ) : null}
        </div>
      )
    }
    case 'raw_markdown':
      return (
        <pre className="my-3 overflow-x-auto rounded-lg border border-dashed border-border bg-surface-muted p-3 font-mono text-[12px] text-muted-foreground">
          {String(content.markdown ?? '')}
        </pre>
      )
    default:
      return (
        <div className="my-2 rounded-md border border-border bg-surface-muted px-3 py-2 font-mono text-[11px] text-muted">
          unsupported block: {block.block_type}
        </div>
      )
  }
}
