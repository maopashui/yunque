import { api } from '@/lib/api'

export function collectImageFiles(data: DataTransfer | null | undefined): File[] {
  if (!data) return []
  const out: File[] = []
  if (data.files?.length) {
    for (const f of Array.from(data.files)) {
      if (f.type.startsWith('image/')) out.push(f)
    }
  }
  if (out.length > 0) return out
  for (const item of Array.from(data.items || [])) {
    if (item.kind === 'file' && item.type.startsWith('image/')) {
      const f = item.getAsFile()
      if (f) out.push(f)
    }
  }
  return out
}

export async function uploadDocumentImage(
  documentId: string,
  file: File,
): Promise<string> {
  const { asset } = await api.uploadDocumentAsset(documentId, file)
  return asset.url
}
