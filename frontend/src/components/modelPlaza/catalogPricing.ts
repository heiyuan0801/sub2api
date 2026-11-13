import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN, BILLING_MODE_VIDEO } from '@/constants/channel'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import { formatScaled, resolveIntervalPrices } from '@/utils/pricing'

type GroupRate = Pick<ModelPlazaGroup, 'rate_multiplier' | 'user_rate_multiplier'> & {
  image_rate_independent?: boolean
  image_rate_multiplier?: number | null
  video_rate_independent?: boolean
  video_rate_multiplier?: number | null
}

export function groupRate(group: GroupRate) {
  return group.user_rate_multiplier ?? group.rate_multiplier
}

export function modelRate(model: PlazaModel, group: GroupRate) {
  if (model.pricing?.billing_mode === BILLING_MODE_IMAGE && group.image_rate_independent) return group.image_rate_multiplier ?? 1
  if (model.pricing?.billing_mode === BILLING_MODE_VIDEO && group.video_rate_independent) return Math.max(0, group.video_rate_multiplier ?? 1)
  return groupRate(group)
}

export function catalogPrices(model: PlazaModel, group: ModelPlazaGroup) {
  const pricing = model.pricing
  const mode = pricing?.billing_mode || BILLING_MODE_TOKEN
  const rate = modelRate(model, group)
  const price = (value?: number | null) => value == null ? null : value * rate
  const firstTier = [...(pricing?.intervals ?? [])].sort((a, b) => a.min_tokens - b.min_tokens)[0]
  const token = firstTier && pricing ? resolveIntervalPrices(firstTier, pricing) : pricing
  return {
    mode,
    input: price(token?.input_price),
    output: price(token?.output_price),
    cache: price(token?.cache_read_price),
    request: price(pricing?.intervals?.find(tier => tier.per_request_price != null)?.per_request_price ?? pricing?.per_request_price),
  }
}

export function formatCatalogPrice(value: number | null, scale = 1) {
  return value == null ? '—' : formatScaled(value, scale, 2)
}
