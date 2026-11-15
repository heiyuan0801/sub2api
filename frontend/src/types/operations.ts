export type NotificationKind =
  | 'price_increase' | 'price_decrease'
  | 'new_plan' | 'plan_unlisted'
  | 'new_group' | 'group_unlisted'
  | 'system' | 'activity'
export type NotificationAudienceMode = 'all' | 'groups' | 'users'

export interface NotificationAudience {
  mode: NotificationAudienceMode
  group_ids?: number[]
  user_ids?: number[]
}

export interface NotificationStats {
  target_count: number
  read_count: number
  unread_count: number
  click_count: number
  delivery_status: string
  attempts: number
  last_error?: string
  published_at?: string
  next_attempt_at?: string
}

export interface NotificationDelivery {
  user_id: number
  email: string
  username: string
  delivered_at: string
  read_at?: string
  click_count: number
}

export const registrationSources = [
  'google', 'bing', 'friend', 'telegram', 'discord', 'github', 'youtube', 'reddit', 'x', 'blog_forum', 'other'
] as const

export type RegistrationSource = typeof registrationSources[number]

export interface RegistrationAttribution {
  selected_source?: RegistrationSource
  source_note?: string
  utm_source?: string
  utm_medium?: string
  utm_campaign?: string
  utm_content?: string
  utm_term?: string
}

export interface RegistrationSourceQuery {
  range: 'today' | '7d' | '30d' | 'custom'
  start_date?: string
  end_date?: string
  dimension: 'selected' | 'utm'
  source?: string
  page?: number
  page_size?: number
}

export interface RegistrationSourceSummary {
  total_registrations: number
  today_registrations: number
  range_registrations: number
  unreported_count: number
  channels: { source: string; count: number; percentage: number }[]
  daily: { date: string; source: string; count: number }[]
}

export interface RegistrationSourceUser extends RegistrationAttribution {
  id: number
  email: string
  username: string
  created_at: string
  signup_source?: string
  inviter_id?: number
  promo_code?: string
}
