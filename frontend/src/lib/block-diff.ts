import i18n from '@/i18n'
import type { Block } from '@/lib/api'

export function blockPlainText(b: Block): string {
  const c = b.content as Record<string, unknown>
  if (typeof c.text === 'string') return c.text
  if (typeof c.markdown === 'string') return c.markdown
  if (Array.isArray(c.inlines)) {
    return (c.inlines as { text?: string }[]).map((i) => i.text ?? '').join('')
  }
  if (Array.isArray(c.items)) {
    return (c.items as { text?: string; inlines?: { text?: string }[] }[])
      .map((item) => {
        if (typeof item.text === 'string') return item.text
        if (Array.isArray(item.inlines)) {
          return item.inlines.map((i) => i.text ?? '').join('')
        }
        return ''
      })
      .join('\n')
  }
  return ''
}

/** Content fingerprint for stable matching across regenerating block ids. */
export function blockFingerprint(b: Block): string {
  return `${b.block_type}\n${blockPlainText(b)}`
}

/** Human-readable row text — never dump raw block_type like "[paragraph]". */
export function blockDisplayText(b: Block): string {
  return blockPlainText(b)
}

function isEmptyBlock(b: Block): boolean {
  return blockPlainText(b).trim() === '' && b.block_type === 'paragraph'
}

/** Drop leading/trailing empty paragraphs so they don't clutter the diff. */
export function trimEmptyParagraphEdges(blocks: Block[]): Block[] {
  let start = 0
  let end = blocks.length
  while (start < end && isEmptyBlock(blocks[start])) start += 1
  while (end > start && isEmptyBlock(blocks[end - 1])) end -= 1
  return blocks.slice(start, end)
}

export type DiffKind = 'ctx' | 'add' | 'del' | 'hunk'

export type DiffRow = {
  key: string
  kind: DiffKind
  text: string
  blockType?: string
  oldNo?: number | null
  newNo?: number | null
  meta?: string
}

/** LCS index pairs between two fingerprint sequences. */
function lcsPairs(a: string[], b: string[]): Array<[number, number]> {
  const n = a.length
  const m = b.length
  const dp: number[][] = Array.from({ length: n + 1 }, () => Array(m + 1).fill(0))
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      if (a[i] === b[j]) dp[i][j] = dp[i + 1][j + 1] + 1
      else dp[i][j] = Math.max(dp[i + 1][j], dp[i][j + 1])
    }
  }
  const pairs: Array<[number, number]> = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      pairs.push([i, j])
      i += 1
      j += 1
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      i += 1
    } else {
      j += 1
    }
  }
  return pairs
}

/**
 * Build GitHub-style unified rows by content fingerprint (not block id),
 * so regenerating UUIDs on materialize won't mark the whole doc as changed.
 */
export function buildUnifiedRows(
  fromBlocks: Block[],
  toBlocks: Block[],
  contextKeep = 3,
): DiffRow[] {
  const from = trimEmptyParagraphEdges(fromBlocks)
  const to = trimEmptyParagraphEdges(toBlocks)
  const fromFP = from.map(blockFingerprint)
  const toFP = to.map(blockFingerprint)
  const pairs = lcsPairs(fromFP, toFP)
  const matchedFrom = new Set(pairs.map(([i]) => i))
  const matchedTo = new Set(pairs.map(([, j]) => j))

  const rows: DiffRow[] = []
  let rowSeq = 0
  let oldNo = 0
  let newNo = 0
  let pi = 0
  let i = 0
  let j = 0

  const push = (row: Omit<DiffRow, 'key'>) => {
    rows.push({ ...row, key: `r-${rowSeq++}` })
  }

  while (i < from.length || j < to.length) {
    const nextPair = pi < pairs.length ? pairs[pi] : null

    // Emit unmatched from (deletes) before next match
    if (i < from.length && (!nextPair || i < nextPair[0])) {
      if (!matchedFrom.has(i)) {
        oldNo += 1
        push({
          kind: 'del',
          text: blockDisplayText(from[i]),
          blockType: from[i].block_type,
          oldNo,
          newNo: null,
        })
      }
      i += 1
      continue
    }

    // Emit unmatched to (adds) before next match
    if (j < to.length && (!nextPair || j < nextPair[1])) {
      if (!matchedTo.has(j)) {
        newNo += 1
        push({
          kind: 'add',
          text: blockDisplayText(to[j]),
          blockType: to[j].block_type,
          oldNo: null,
          newNo,
        })
      }
      j += 1
      continue
    }

    // Matched pair → context
    if (nextPair && i === nextPair[0] && j === nextPair[1]) {
      oldNo += 1
      newNo += 1
      push({
        kind: 'ctx',
        text: blockDisplayText(to[j]),
        blockType: to[j].block_type,
        oldNo,
        newNo,
      })
      i += 1
      j += 1
      pi += 1
      continue
    }

    // Safety advance
    if (i < from.length) i += 1
    else if (j < to.length) j += 1
    else break
  }

  return collapseContext(rows, contextKeep)
}

function collapseContext(rows: DiffRow[], keep: number): DiffRow[] {
  const out: DiffRow[] = []
  let i = 0
  let seq = 0
  while (i < rows.length) {
    if (rows[i].kind !== 'ctx') {
      out.push(rows[i])
      i += 1
      continue
    }
    let j = i
    while (j < rows.length && rows[j].kind === 'ctx') j += 1
    const run = rows.slice(i, j)
    if (run.length <= keep * 2 + 1) {
      out.push(...run)
    } else {
      out.push(...run.slice(0, keep))
      const hidden = run.length - keep * 2
      const firstHidden = run[keep]
      const lastHidden = run[run.length - keep - 1]
      out.push({
        key: `hunk-${seq++}`,
        kind: 'hunk',
        text: i18n.t('blockDiff.unchangedHunk', { count: hidden }),
        meta: `@@ -${firstHidden.oldNo ?? 0},${hidden} +${firstHidden.newNo ?? 0},${hidden} @@`,
        oldNo: firstHidden.oldNo,
        newNo: lastHidden.newNo,
      })
      out.push(...run.slice(-keep))
    }
    i = j
  }
  return out
}

export function summarizeRows(rows: DiffRow[]) {
  let additions = 0
  let deletions = 0
  for (const r of rows) {
    if (r.kind === 'add') additions += 1
    if (r.kind === 'del') deletions += 1
  }
  return { additions, deletions }
}
