<template>
  <section v-if="providers.length" class="gateway-vendors" aria-labelledby="gateway-vendors-title">
    <div class="feature-shell">
      <h2 id="gateway-vendors-title">{{ t('publicSite.market.providers') }}</h2>
      <ul>
        <li v-for="provider in providers.slice(0, 8)" :key="provider.name">
          <RouterLink :to="modelPath(provider.name)">
            <PlatformIcon :platform="provider.name as GroupPlatform" size="lg" aria-hidden="true" />
            <span>{{ platformLabel(provider.name) }}</span>
          </RouterLink>
        </li>
      </ul>
    </div>
  </section>

  <section id="features" class="gateway-features" aria-labelledby="gateway-features-title">
    <div class="feature-shell">
      <h2 id="gateway-features-title" class="section-heading">{{ t('publicHome.services.title') }}</h2>
      <div class="primary-features">
        <RouterLink
          v-for="feature in primaryFeatures"
          :key="feature.title"
          :to="feature.to"
          class="feature-card feature-card-large"
        >
          <span class="feature-icon"><Icon :name="feature.icon" size="xl" aria-hidden="true" /></span>
          <h3>{{ feature.title }}</h3>
          <p>{{ feature.description }}</p>
          <span class="feature-link">{{ copy.learnMore }} <Icon name="arrowRight" size="sm" aria-hidden="true" /></span>
        </RouterLink>
      </div>
      <div class="secondary-features">
        <RouterLink
          v-for="feature in secondaryFeatures"
          :key="feature.title"
          :to="feature.to"
          class="feature-card"
        >
          <span class="feature-icon"><Icon :name="feature.icon" size="md" aria-hidden="true" /></span>
          <h3>{{ feature.title }}</h3>
          <p>{{ feature.description }}</p>
          <span class="feature-link">{{ copy.learnMore }} <Icon name="arrowRight" size="sm" aria-hidden="true" /></span>
        </RouterLink>
      </div>
    </div>
  </section>

  <section id="controls" class="gateway-controls" aria-labelledby="gateway-controls-title">
    <div class="feature-shell">
      <header class="section-header control-header">
        <h2 id="gateway-controls-title" class="section-heading">{{ copy.controlTitle }} <em>{{ copy.controlAccent }}</em></h2>
        <p>{{ copy.controlDescription }}</p>
      </header>
      <div class="control-grid">
        <article v-for="(control, index) in controls" :key="control.id" class="control-card">
          <div class="control-caption">
            <span>{{ String(index + 1).padStart(2, '0') }} / {{ control.label }}</span>
            <i aria-hidden="true"></i>
            <small>{{ copy.sample }}</small>
          </div>
          <h3>{{ control.title }}</h3>
          <p>{{ control.description }}</p>
          <ul v-if="control.id === 'limits'" class="control-progress">
            <li v-for="item in quotas" :key="item.name">
              <div><code>{{ item.name }}</code><strong>{{ item.percent }}%</strong></div>
              <progress :value="item.percent" max="100" :aria-label="item.name"></progress>
            </li>
          </ul>
          <ul v-else class="control-data">
            <li v-for="[label, value] in control.rows" :key="label">
              <code>{{ label }}</code><span>{{ value }}</span>
            </li>
          </ul>
        </article>
      </div>
      <div class="control-setup">
        <div>
          <span class="control-caption">06 / {{ copy.setup }}</span>
          <h3>{{ copy.setupTitle }}</h3>
          <p>{{ copy.setupDescription }}</p>
        </div>
        <div class="setup-options">
          <RouterLink v-for="(option, index) in setupOptions" :key="option.title" :to="option.to">
            <span>{{ String(index + 1).padStart(2, '0') }}</span>
            <strong>{{ option.title }}</strong>
            <small>{{ option.description }}</small>
          </RouterLink>
        </div>
      </div>
      <div class="control-actions">
        <RouterLink to="/keys" class="control-cta">{{ t('publicHome.quickstart.createKey.title') }} <Icon name="arrowRight" size="sm" aria-hidden="true" /></RouterLink>
        <RouterLink :to="docsPath">{{ t('publicSite.nav.apiDocs') }} <Icon name="externalLink" size="sm" aria-hidden="true" /></RouterLink>
      </div>
    </div>
  </section>

  <section id="products" class="gateway-products" aria-labelledby="gateway-products-title">
    <div class="feature-shell">
      <header class="section-header products-header">
        <h2 id="gateway-products-title" class="section-heading">{{ copy.productsTitle }} <em>{{ copy.productsAccent }}</em></h2>
        <p>{{ copy.productsDescription }}</p>
      </header>
      <div class="product-grid">
        <article v-for="product in products" :key="product.id" class="product-ticket" :data-product="product.id">
          <header class="product-copy">
            <span class="product-kicker"><i aria-hidden="true"></i>{{ product.category }}</span>
            <h3>{{ product.title }}</h3>
            <p>{{ product.description }}</p>
          </header>
          <TabsRoot
            v-slot="{ modelValue }"
            :model-value="selectedScreens[product.id] || product.screens[0].id"
            class="product-preview"
            @update:model-value="selectedScreens[product.id] = String($event)"
          >
            <div class="preview-controls">
              <TabsList :aria-label="product.title" class="preview-tabs">
                <TabsTrigger v-for="screen in product.screens" :key="screen.id" :value="screen.id">{{ screen.tab }}</TabsTrigger>
              </TabsList>
              <div class="preview-arrows">
                <span>{{ screenIndex(product, modelValue) + 1 }}/{{ product.screens.length }}</span>
                <button type="button" :aria-label="copy.prev" @click="stepScreen(product, modelValue, -1)"><Icon name="chevronLeft" size="xs" aria-hidden="true" /></button>
                <button type="button" :aria-label="copy.next" @click="stepScreen(product, modelValue, 1)"><Icon name="chevronRight" size="xs" aria-hidden="true" /></button>
              </div>
            </div>

            <TabsContent v-for="screen in product.screens" :key="screen.id" :value="screen.id" class="preview-screen">
              <div class="preview-heading">
                <h4>{{ screen.title || screen.tab }}</h4>
                <span v-if="screen.sample">{{ copy.sample }}</span>
              </div>
              <div v-if="screen.bars" class="preview-chart" aria-hidden="true">
                <i v-for="(bar, position) in screen.bars" :key="position" :style="{ height: bar + '%' }"></i>
              </div>
              <p v-if="screen.code" class="preview-code"><code>{{ screen.code }}</code></p>
              <div v-if="screen.headers" class="preview-scroll">
                <table>
                  <thead><tr><th v-for="label in screen.headers" :key="label" scope="col">{{ label }}</th></tr></thead>
                  <tbody>
                    <tr v-for="(row, rowIndex) in screen.rows" :key="rowIndex">
                      <td v-for="(cell, cellIndex) in row" :key="cellIndex">{{ cell }}</td>
                    </tr>
                  </tbody>
                </table>
                <p v-if="!screen.rows?.length" class="preview-empty">{{ t('common.noData') }}</p>
              </div>
            </TabsContent>
          </TabsRoot>
          <footer class="ticket-footer">
            <span aria-hidden="true" class="ticket-mark">{{ product.mark }} <i></i></span>
            <RouterLink :to="product.to">{{ product.action }} <Icon name="externalLink" size="xs" aria-hidden="true" /></RouterLink>
          </footer>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from 'reka-ui'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { catalogPrices, formatCatalogPrice, groupRate } from '@/components/modelPlaza/catalogPricing'
import { BILLING_MODE_TOKEN } from '@/constants/channel'
import { localizedPublicPath } from '@/i18n/localeConfig'
import type { GroupPlatform } from '@/types'
import { platformLabel } from '@/utils/platformColors'
import { gatewayFeatureCopy } from './gatewayFeatureCopy'

const props = defineProps<{
  siteName: string
  providers: { name: string; count: number }[]
  groups: ModelPlazaGroup[]
  showModelPlaza: boolean
  showMonitor: boolean
}>()

const { locale, t } = useI18n()
const copy = computed(() => gatewayFeatureCopy[locale.value] ?? gatewayFeatureCopy.en)
const docsPath = computed(() => localizedPublicPath('/docs/api', locale.value))
const marketPath = computed(() => localizedPublicPath('/model-plaza', locale.value))
const quickStartPath = computed(() => localizedPublicPath('/home#quick-start', locale.value))

function modelPath(platform: string) {
  return localizedPublicPath(`/model-plaza?${new URLSearchParams({ search: platformLabel(platform) })}`, locale.value)
}

const primaryFeatures = computed(() => [
  { icon: 'cube', title: t('publicHome.services.unifiedApi.title'), description: t('publicHome.services.unifiedApi.description'), to: quickStartPath.value },
  { icon: 'grid', title: t('publicHome.services.modelSelection.title'), description: t('publicHome.services.modelSelection.description'), to: props.showModelPlaza ? marketPath.value : docsPath.value },
  {
    icon: 'chartBar',
    title: props.showMonitor ? copy.value.performanceTitle : t('publicHome.services.usage.title'),
    description: props.showMonitor ? copy.value.performanceDescription : t('publicHome.services.usage.description'),
    to: props.showMonitor ? '/monitor' : '/usage',
  },
] as const)
const secondaryFeatures = computed(() => [
  { icon: 'shield', title: t('publicHome.services.keys.title'), description: t('publicHome.services.keys.description'), to: '/keys' },
  { icon: 'creditCard', title: t('publicHome.services.costs.title'), description: t('publicHome.services.costs.description'), to: '/usage' },
  { icon: 'database', title: copy.value.cacheTitle, description: copy.value.cacheDescription, to: '/usage' },
  { icon: 'sparkles', title: t('publicHome.services.multimodal.title'), description: t('publicHome.services.multimodal.description'), to: props.showModelPlaza ? marketPath.value : docsPath.value },
  { icon: 'key', title: t('publicSite.footer.usage'), description: copy.value.keyUsageDescription, to: '/key-usage' },
  { icon: 'swap', title: copy.value.routingTitle, description: copy.value.routingDescription, to: '/keys' },
  { icon: 'terminal', title: t('publicHome.quickstart.title'), description: t('publicHome.quickstart.description'), to: quickStartPath.value },
  { icon: 'book', title: t('publicSite.nav.apiDocs'), description: t('publicHome.quickstart.connectApp.description'), to: docsPath.value },
] as const)
const quotas = [
  { name: 'production-api', percent: 72 },
  { name: 'support-bot', percent: 91 },
  { name: 'staging', percent: 34 },
]
const controls = computed(() => [
  { id: 'access', label: copy.value.access, title: copy.value.accessTitle, description: copy.value.accessDescription, rows: [['production-api', 'sk-••••••••a2f1'], ['support-bot', 'sk-••••••••f627'], ['staging', 'sk-••••••••c181']] },
  { id: 'rules', label: copy.value.rules, title: copy.value.rulesTitle, description: copy.value.rulesDescription, rows: [['production-api', '192.0.2.0/24'], ['support-bot', '198.51.100.0/24'], ['staging', '203.0.113.0/24']] },
  { id: 'limits', label: copy.value.limits, title: copy.value.limitsTitle, description: copy.value.limitsDescription, rows: [] },
  { id: 'routing', label: copy.value.routing, title: copy.value.routingTitle, description: copy.value.routingDescription, rows: [['01', 'primary'], ['02', 'secondary'], ['03', 'backup']] },
  { id: 'logs', label: copy.value.requestLog, title: copy.value.requestLogTitle, description: copy.value.requestLogDescription, rows: [['POST /v1/responses', '200'], ['POST /v1/messages', '200'], ['GET /v1/models', '200']] },
])
const setupOptions = computed(() => [
  { title: 'OpenAI SDK', description: 'baseURL · /v1', to: quickStartPath.value },
  { title: 'Claude Code', description: 'ANTHROPIC_BASE_URL', to: docsPath.value },
  { title: 'cURL', description: 'POST /v1/chat/completions', to: quickStartPath.value },
])

interface PreviewScreen {
  id: string
  tab: string
  title?: string
  sample?: boolean
  headers?: string[]
  rows?: string[][]
  bars?: number[]
  code?: string
}
interface Product {
  id: string
  category: string
  title: string
  description: string
  mark: string
  action: string
  to: string
  screens: PreviewScreen[]
}
const selectedScreens = reactive<Record<string, string>>({})

const publicPrices = computed(() => props.groups.flatMap(group =>
  group.models.flatMap(model => {
    const price = catalogPrices(model, group)
    return price.mode === BILLING_MODE_TOKEN ? [[model.name, formatCatalogPrice(price.input, 1_000_000), formatCatalogPrice(price.output, 1_000_000)]] : []
  })
).slice(0, 6))
const products = computed<Product[]>(() => {
  const labels = copy.value
  const result: Product[] = [
    {
      id: 'gateway', category: labels.access, title: props.siteName, description: t('publicHome.services.unifiedApi.description'),
      mark: 'A', action: t('nav.apiKeys'), to: '/keys',
      screens: [
        { id: 'keys', tab: t('nav.apiKeys'), sample: true, headers: [t('common.name'), t('nav.apiKeys'), t('common.status')], rows: [['production-api', 'sk-••••••••a2f1', labels.active], ['support-bot', 'sk-••••••••f627', labels.active], ['staging', 'sk-••••••••c181', labels.active], ['assistant', 'sk-••••••••20c5', labels.active], ['reports', 'sk-••••••••a947', labels.active]] },
        { id: 'limits', tab: labels.limitsTab, title: labels.limitsTitle, sample: true, headers: [t('common.name'), labels.limitsTab, '%'], rows: [['production-api', '$50.00', '72%'], ['support-bot', '$25.00', '91%'], ['staging', '$15.00', '34%']] },
      ],
    },
    {
      id: 'usage', category: labels.requestLog, title: t('publicHome.services.usage.title'), description: t('publicHome.services.usage.description'),
      mark: 'B', action: t('nav.usage'), to: '/usage',
      screens: [
        { id: 'requests', tab: labels.requestsTab, sample: true, headers: [t('publicHome.examples.request'), 'Token', t('publicSite.performance.averageLatency')], rows: [['POST /v1/responses', '3.4K', '1.20 s'], ['POST /v1/chat/completions', '12.9K', '0.53 s'], ['POST /v1/messages', '14.1K', '1.84 s']] },
        { id: 'cache', tab: labels.cachedTab, sample: true, bars: [38, 45, 42, 67, 55, 80, 68, 79, 65, 92, 87, 98, 83, 91] },
      ],
    },
  ]
  if (props.showModelPlaza) {
    result.push({
      id: 'models', category: t('publicSite.market.providers'), title: t('publicSite.market.heading'), description: t('publicSite.market.featuredDescription'),
      mark: 'C', action: t('publicSite.market.explore'), to: marketPath.value,
      screens: [
        { id: 'models', tab: t('publicSite.market.availableModels'), headers: [t('publicSite.modelDetail.modelId'), t('publicSite.market.input'), t('publicSite.market.output')], rows: publicPrices.value },
        { id: 'groups', tab: t('publicSite.market.groups'), headers: [t('publicSite.models.group'), t('publicSite.models.rate')], rows: props.groups.slice(0, 6).map(group => [group.name, `${groupRate(group)}×`]) },
      ],
    })
  }
  if (props.showMonitor) {
    result.push({
      id: 'status', category: labels.performanceTitle, title: t('nav.channelStatus'), description: labels.performanceDescription,
      mark: 'D', action: t('nav.channelStatus'), to: '/monitor',
      screens: [
        { id: 'status', tab: t('publicSite.performance.successRate'), sample: true, bars: [89, 94, 68, 92, 95, 87, 97, 97, 81, 96, 92, 91, 94, 97] },
        { id: 'latency', tab: labels.latencyTab, sample: true, bars: [64, 46, 92, 62, 58, 44, 68, 57, 78, 46, 57, 51, 42, 54] },
      ],
    })
  }
  result.push({
    id: 'docs', category: labels.setup, title: t('publicSite.nav.apiDocs'), description: t('publicHome.quickstart.connectApp.description'),
    mark: 'E', action: t('publicHome.actions.documentation'), to: docsPath.value,
    screens: [
      { id: 'chat', tab: t('publicHome.examples.protocols.chat'), code: 'POST /v1/chat/completions\nContent-Type: application/json\n\n{ "model": "MODEL_ID", "messages": [ … ] }' },
      { id: 'responses', tab: t('publicHome.examples.protocols.responses'), code: 'POST /v1/responses\nContent-Type: application/json\n\n{ "model": "MODEL_ID", "input": "Hello!" }' },
    ],
  })
  return result
})

function screenIndex(product: Product, value: unknown) {
  return Math.max(0, product.screens.findIndex(screen => screen.id === value))
}
function stepScreen(product: Product, value: unknown, offset: number) {
  const length = product.screens.length
  const next = (screenIndex(product, value) + offset + length) % length
  selectedScreens[product.id] = product.screens[next].id
}
</script>

<style scoped>
.feature-shell {
  width: min(100% - clamp(32px, 5.8vw, 80px), 1280px);
  margin-inline: auto;
}

.gateway-vendors {
  padding-block: 72px 80px;
  color: var(--gateway-muted);
}
.gateway-vendors h2 {
  text-align: center;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: .12em;
  text-transform: uppercase;
}
.gateway-vendors ul {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: clamp(20px, 4.4vw, 65px);
  margin-top: 40px;
}
.gateway-vendors a {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: var(--gateway-ink);
  font-size: clamp(21px, 2.45vw, 30px);
  font-weight: 650;
  letter-spacing: -.04em;
  opacity: .74;
}
.gateway-vendors a:hover { opacity: 1; }

.gateway-features,
.gateway-controls,
.gateway-products {
  scroll-margin-top: 90px;
  padding-block: clamp(62px, 9vw, 128px);
}
.gateway-features {
  background-image: radial-gradient(var(--gateway-line) .6px, transparent .6px);
  background-position: center;
  background-size: 34px 34px;
}
.section-heading {
  color: var(--gateway-ink);
  font-family: var(--gateway-display, inherit);
  font-size: clamp(34px, 4vw, 48px);
  font-weight: 700;
  letter-spacing: -.045em;
  line-height: 1.08;
  text-wrap: balance;
}
.section-heading em { color: var(--gateway-muted); font-style: normal; }
.gateway-features .section-heading { max-width: 670px; margin-bottom: 62px; }
.primary-features,
.secondary-features {
  display: grid;
  gap: 18px;
}
.primary-features { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.secondary-features { grid-template-columns: repeat(4, minmax(0, 1fr)); margin-top: 20px; }
.feature-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-height: 227px;
  padding: 23px;
  border: 1px solid var(--gateway-line);
  border-radius: 17px;
  background: var(--gateway-surface);
  box-shadow: 0 1px 2px rgb(0 0 0 / 4%);
  transition: border-color .2s, box-shadow .2s;
}
.feature-card:hover {
  border-color: color-mix(in srgb, var(--gateway-line) 45%, var(--gateway-accent));
  box-shadow: 0 5px 22px rgb(0 0 0 / 6%);
}
.feature-card-large {
  min-height: 363px;
  padding: 32px;
  border-radius: 20px;
  outline: 1px solid color-mix(in srgb, var(--gateway-line) 48%, transparent);
  outline-offset: 7px;
  margin: 8px;
}
.feature-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 43px;
  height: 43px;
  margin-bottom: 16px;
  border-radius: 10px;
  color: var(--gateway-muted);
  background: var(--gateway-soft);
  flex: none;
}
.feature-card-large .feature-icon {
  width: 73px;
  height: 73px;
  border-radius: 13px;
  margin-bottom: clamp(30px, 4.2vw, 55px);
  background: color-mix(in srgb, var(--gateway-soft) 72%, var(--gateway-line));
}
.feature-card-large:nth-child(2) .feature-icon {
  background: color-mix(in srgb, var(--gateway-accent) 12%, var(--gateway-surface));
  color: var(--gateway-accent);
}
.feature-card-large:nth-child(3) .feature-icon {
  background: color-mix(in srgb, #30a46c 12%, var(--gateway-surface));
  color: #25976e;
}
.feature-card h3 {
  color: var(--gateway-ink);
  font-size: 18px;
  font-weight: 650;
  letter-spacing: -.025em;
  line-height: 1.22;
}
.feature-card-large h3 {
  font-family: var(--gateway-display, inherit);
  font-size: clamp(22px, 2.4vw, 30px);
  font-weight: 700;
  letter-spacing: -.045em;
}
.feature-card p {
  margin: 10px 0 22px;
  color: var(--gateway-muted);
  font-size: 14px;
  line-height: 1.6;
}
.feature-card-large p { font-size: 16px; }
.feature-link,
.control-actions a {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--gateway-ink);
  font-size: 14px;
  font-weight: 550;
}
.feature-link { margin-top: auto; }

.section-header p {
  max-width: 760px;
  color: var(--gateway-muted);
  font-size: clamp(16px, 1.47vw, 18px);
  line-height: 1.55;
}
.control-header {
  display: grid;
  grid-template-columns: 1.43fr 1fr;
  align-items: end;
  gap: 32px;
  margin-bottom: 54px;
}
.control-header h2 { text-wrap: initial; }
.control-header em { display: block; }
.control-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 16px;
}
.control-card {
  grid-column: span 2;
  min-width: 0;
  padding: clamp(18px, 2.3vw, 28px);
  border: 1px solid var(--gateway-line);
  border-radius: 16px;
  background: var(--gateway-surface);
}
.control-card:nth-child(-n+2) { grid-column: span 3; }
.control-card:nth-child(2) {
  border-color: color-mix(in srgb, var(--gateway-accent) 25%, var(--gateway-line));
}
.control-caption,
.product-kicker {
  display: flex;
  align-items: center;
  gap: 11px;
  color: var(--gateway-muted);
  font: 11px/1.4 ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: .14em;
  text-transform: uppercase;
}
.control-caption i { flex: 1; height: 1px; background: var(--gateway-line); }
.control-caption small {
  flex: none;
  border-radius: 999px;
  padding: 3px 8px;
  background: color-mix(in srgb, var(--gateway-accent) 10%, var(--gateway-surface));
  color: var(--gateway-accent);
  font-size: 10px;
  letter-spacing: .05em;
}
.control-card h3,
.control-setup h3 {
  margin-top: 20px;
  color: var(--gateway-ink);
  font-family: var(--gateway-display, inherit);
  font-size: 21px;
  font-weight: 700;
  letter-spacing: -.035em;
  line-height: 1.24;
}
.control-card p,
.control-setup p {
  margin-top: 9px;
  color: var(--gateway-muted);
  font-size: 14px;
  line-height: 1.6;
}
.control-data,
.control-progress {
  margin-top: 24px;
  border: 1px solid var(--gateway-line);
  border-radius: 13px;
  color: var(--gateway-ink);
}
.control-data li,
.control-progress li {
  padding: 10px 14px;
  font-size: 12px;
}
.control-data li + li,
.control-progress li + li {
  border-top: 1px solid var(--gateway-line);
}
.control-data li,
.control-progress li div {
  display: flex;
  justify-content: space-between;
  gap: 9px;
}
.control-data li code { overflow-wrap: anywhere; }
.control-data li span { color: var(--gateway-muted); }
.control-progress progress {
  display: block;
  width: 100%;
  height: 5px;
  margin-top: 5px;
  border: 0;
  accent-color: var(--gateway-accent);
}
.control-progress progress::-webkit-progress-bar { border-radius: 999px; background: var(--gateway-soft); }
.control-progress progress::-webkit-progress-value { border-radius: 999px; background: var(--gateway-accent); }
.control-setup {
  display: grid;
  align-items: center;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.42fr);
  gap: clamp(22px, 4vw, 50px);
  margin-top: 16px;
  padding: clamp(22px, 3vw, 40px);
  border: 1px solid var(--gateway-line);
  border-radius: 16px;
  background: radial-gradient(ellipse at top right, color-mix(in srgb, var(--gateway-accent) 8%, transparent), transparent 70%), var(--gateway-surface);
}
.control-setup h3 { font-size: clamp(27px, 3.2vw, 36px); }
.control-setup p { font-size: 16px; }
.setup-options {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}
.setup-options a {
  display: flex;
  flex-direction: column;
  gap: 7px;
  min-height: 134px;
  padding: 18px 15px;
  border: 1px solid var(--gateway-line);
  border-radius: 14px;
  background: color-mix(in srgb, var(--gateway-surface) 84%, transparent);
  color: var(--gateway-ink);
  overflow-wrap: anywhere;
}
.setup-options a:hover { border-color: var(--gateway-accent); }
.setup-options span,
.setup-options small { color: var(--gateway-muted); font-size: 12px; }
.setup-options span { margin-bottom: 5px; font-family: ui-monospace, monospace; }
.control-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 22px;
  margin-top: 32px;
}
.control-actions .control-cta {
  padding: 12px 22px;
  border-radius: 999px;
  color: white;
  background: var(--gateway-accent);
  box-shadow: 0 5px 14px color-mix(in srgb, var(--gateway-accent) 26%, transparent);
}

.gateway-products {
  border-block: 1px solid var(--gateway-line);
  background: var(--gateway-soft);
}
.products-header { margin-bottom: 54px; }
.products-header p { margin-top: 19px; }
.product-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}
.product-ticket {
  --ticket-color: #bf9bf1;
  --ticket-line: color-mix(in srgb, var(--ticket-color) 36%, var(--gateway-line));
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--gateway-line);
  border-radius: 16px;
  background: var(--gateway-surface);
}
.product-ticket[data-product='gateway'] {
  --ticket-color: #9fbff1;
  grid-column: 1 / -1;
  grid-template-columns: minmax(0, 5fr) minmax(0, 7fr);
  align-items: stretch;
}
.product-ticket[data-product='models'] { --ticket-color: #96c8ab; }
.product-ticket[data-product='status'] { --ticket-color: #ebad87; }
.product-ticket[data-product='docs'] { --ticket-color: #95b0ee; }
.product-copy {
  min-width: 0;
  align-self: center;
  padding: clamp(24px, 3.4vw, 42px);
}
.product-ticket[data-product='gateway'] .product-copy { padding-block: 64px; }
.product-kicker i {
  flex: none;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--ticket-color);
}
.product-copy h3 {
  margin-top: 17px;
  color: var(--gateway-ink);
  font-family: var(--gateway-display, inherit);
  font-size: clamp(28px, 3.75vw, 48px);
  font-weight: 700;
  letter-spacing: -.05em;
  line-height: 1.04;
  overflow-wrap: anywhere;
}
.product-ticket:not([data-product='gateway']) .product-copy h3 {
  font-size: clamp(26px, 3vw, 38px);
}
.product-copy p {
  margin-top: 12px;
  max-width: 48ch;
  color: var(--gateway-muted);
  font-size: 17px;
  line-height: 1.55;
}
.product-preview {
  min-width: 0;
  padding: clamp(16px, 2.6vw, 32px);
  padding-bottom: 16px;
  background: radial-gradient(ellipse at 50% 0%, color-mix(in srgb, var(--ticket-color) 28%, var(--gateway-soft)), var(--gateway-soft) 74%);
}
.product-ticket[data-product='gateway'] .product-preview { border-inline-start: 1px solid var(--gateway-line); }
.product-ticket:not([data-product='gateway']) .product-preview { grid-row: 1; }
.preview-controls,
.preview-arrows,
.preview-tabs {
  display: flex;
  align-items: center;
  gap: 6px;
}
.preview-controls { justify-content: space-between; margin-bottom: 12px; }
.preview-tabs { flex-wrap: wrap; gap: 5px; }
.preview-arrows { flex: none; }
.preview-arrows span {
  margin-inline-end: 4px;
  color: var(--gateway-muted);
  font: 11px ui-monospace, monospace;
  white-space: nowrap;
}
.preview-tabs button,
.preview-arrows button {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px 9px;
  border: 1px solid var(--gateway-line);
  border-radius: 999px;
  color: var(--gateway-muted);
  background: var(--gateway-surface);
  font-size: 12px;
  line-height: 1.3;
}
.preview-arrows button { width: 27px; height: 27px; padding: 0; }
.preview-tabs button[data-state='active'] {
  border-color: var(--ticket-line);
  color: var(--gateway-ink);
  background: color-mix(in srgb, var(--ticket-color) 24%, var(--gateway-surface));
}
.preview-screen {
  min-height: 185px;
  overflow: hidden;
  border: 1px solid var(--gateway-line);
  border-radius: 12px;
  color: var(--gateway-ink);
  background: var(--gateway-surface);
}
.product-ticket[data-product='gateway'] .preview-screen { min-height: 263px; }
.preview-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 9px;
  padding: 14px;
}
.preview-heading h4 { font-size: 14px; font-weight: 700; }
.preview-heading span {
  color: var(--gateway-muted);
  font: 10px/1.3 ui-monospace, monospace;
  text-align: end;
  text-transform: uppercase;
}
.preview-scroll { overflow-x: auto; }
.preview-screen table { width: 100%; font-size: 11px; }
.preview-screen th,
.preview-screen td {
  padding: 9px 12px;
  border-top: 1px solid var(--gateway-line);
  text-align: start;
  white-space: nowrap;
}
.preview-screen th {
  color: var(--gateway-muted);
  background: var(--gateway-soft);
  font-weight: 600;
}
.preview-empty { padding: 16px 12px; color: var(--gateway-muted); font-size: 12px; }
.preview-chart {
  display: flex;
  align-items: flex-end;
  justify-content: stretch;
  gap: 5px;
  height: 119px;
  margin: 0 14px 18px;
  padding: 10px 11px 0;
  border-bottom: 1px solid var(--gateway-line);
  background: repeating-linear-gradient(to bottom, var(--gateway-line) 0 1px, transparent 1px 29px);
}
.preview-chart i {
  flex: 1;
  border-radius: 3px 3px 0 0;
  background: color-mix(in srgb, var(--ticket-color) 80%, var(--gateway-accent));
}
.preview-code {
  margin: 8px 14px 14px;
  padding: 15px;
  border: 1px solid var(--gateway-line);
  border-radius: 8px;
  background: var(--gateway-soft);
  color: var(--gateway-ink);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.ticket-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  grid-column: 1 / -1;
  padding: 20px 28px;
  background: radial-gradient(circle, color-mix(in srgb, var(--gateway-muted) 42%, transparent) 2px, transparent 2.8px) 28px 0 / 11px 8px repeat-x;
}
.ticket-footer a {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--gateway-accent);
  font-size: 14px;
  font-weight: 600;
}
.ticket-mark {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: var(--gateway-muted);
  font: 36px/1 Georgia, serif;
}
.ticket-mark i {
  width: 78px;
  height: 24px;
  background: repeating-linear-gradient(to right, currentColor 0 1px, transparent 1px 3px, currentColor 3px 5px, transparent 5px 7px, currentColor 7px 8px, transparent 8px 10px);
  opacity: .5;
}
.feature-shell :is(a, button):focus-visible {
  outline: 2px solid var(--gateway-accent);
  outline-offset: 3px;
}
@media (max-width: 840px) {
  .feature-card-large { min-height: 306px; padding: 24px; }
  .secondary-features { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .control-header, .control-setup { grid-template-columns: 1fr; }
  .control-header, .products-header { margin-bottom: 34px; }
  .control-card { grid-column: span 3; }
  .product-ticket[data-product='gateway'] { grid-template-columns: 1fr; }
  .product-ticket[data-product='gateway'] .product-copy { padding-block: 28px; }
  .product-ticket[data-product='gateway'] .product-preview { border-inline-start: 0; padding-top: 18px; }
}
@media (max-width: 580px) {
  .gateway-vendors { padding-block: 48px; }
  .gateway-vendors ul { margin-top: 25px; }
  .gateway-vendors a { gap: 7px; }
  .gateway-vendors svg { width: 23px; height: 23px; }
  .gateway-features .section-heading { margin-bottom: 32px; }
  .primary-features, .secondary-features, .product-grid, .control-grid { grid-template-columns: minmax(0, 1fr); }
  .control-grid .control-card, .product-ticket[data-product='gateway'] { grid-column: auto; }
  .feature-card, .feature-card-large { min-height: 0; }
  .feature-card-large { margin: 4px; outline-offset: 4px; }
  .feature-card-large .feature-icon { width: 54px; height: 54px; margin-bottom: 24px; }
  .setup-options { grid-template-columns: 1fr; }
  .setup-options a { min-height: 0; }
  .ticket-footer { padding-inline: 17px; background-position: 16px 0; }
  .ticket-mark i { width: 55px; }
}
@media (prefers-reduced-motion: reduce) {
  .feature-card { transition: none; }
}
</style>
