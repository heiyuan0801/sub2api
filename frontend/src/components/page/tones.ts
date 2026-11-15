export type PageTone = 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'brand'

export const toneText: Record<PageTone, string> = {
  neutral: 'text-muted-foreground',
  success: 'text-success',
  warning: 'text-warning',
  danger: 'text-danger',
  info: 'text-info',
  brand: 'text-brand'
}

export const toneDot: Record<PageTone, string> = {
  neutral: 'bg-muted-foreground ring-muted-foreground/15',
  success: 'bg-success ring-success/15',
  warning: 'bg-warning ring-warning/15',
  danger: 'bg-danger ring-danger/15',
  info: 'bg-info ring-info/15',
  brand: 'bg-brand ring-brand/15'
}
