export const groupDisplayTags = ['recommended', 'popular', 'new'] as const

export type GroupDisplayTag = (typeof groupDisplayTags)[number]

export const customGroupDisplayTagColors = ['blue', 'violet', 'emerald', 'orange', 'rose', 'slate'] as const
export const customGroupDisplayTagIcons = ['tag', 'star', 'flame', 'sparkles', 'bolt', 'shield', 'gem'] as const
export const customGroupDisplayTagLimit = 5
export const customGroupDisplayTagLabelLimit = 20

export interface CustomGroupDisplayTag {
  label: string
  color: (typeof customGroupDisplayTagColors)[number]
  icon: (typeof customGroupDisplayTagIcons)[number]
}

export function customGroupDisplayTagLabelError(tags: readonly CustomGroupDisplayTag[], index: number) {
  const raw = tags[index]!.label
  const label = raw.trim()
  if (!label) return 'required'
  if (Array.from(label).length > customGroupDisplayTagLabelLimit) return 'length'
  if (/[\p{Cc}\p{Bidi_Control}]/u.test(raw)) return 'characters'
  const folded = label.toLowerCase()
  if (tags.some((tag, position) => position !== index && tag.label.trim().toLowerCase() === folded)) {
    return 'duplicate'
  }
}
