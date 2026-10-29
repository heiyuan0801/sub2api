// @vitest-environment node
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import viteConfig from '../../../../vite.config'

const frontendRoot = resolve(__dirname, '../../../..')
const stripeConsumers = [
  'src/views/user/StripePaymentView.vue',
  'src/views/user/StripePopupView.vue',
  'src/components/payment/StripePaymentInline.vue',
]

function readFrontendFile(path: string): string {
  return readFileSync(resolve(frontendRoot, path), 'utf8')
}

describe('Stripe lazy-loading contract', () => {
  it.each(stripeConsumers)('%s uses the side-effect-free Stripe loader', (path) => {
    const source = readFrontendFile(path)

    expect(source).toContain("await import('@stripe/stripe-js/pure')")
    expect(source).not.toMatch(/await import\(['"]@stripe\/stripe-js['"]\)/)
  })

  it('keeps Stripe out of the shared vendor chunk', async () => {
    const config = await (typeof viteConfig === 'function'
      ? viteConfig({ command: 'build', mode: 'production' })
      : viteConfig)
    const output = config.build?.rollupOptions?.output
    if (!output || Array.isArray(output) || typeof output.manualChunks !== 'function') {
      throw new Error('Expected a manual chunk function')
    }
    const manualChunks = output.manualChunks as (id: string) => string | undefined
    expect(manualChunks('/app/node_modules/@stripe/stripe-js/dist/pure.js')).toBe('vendor-stripe')
    expect(manualChunks('/app/node_modules/vue/dist/vue.runtime.esm-bundler.js')).toBeUndefined()
  })
})
