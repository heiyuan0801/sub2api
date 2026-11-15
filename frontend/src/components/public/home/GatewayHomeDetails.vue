<template>
  <section v-if="showModels" id="routing" class="gateway-routing gateway-band" aria-labelledby="gateway-routing-title">
    <div class="detail-shell">
      <div class="routing-heading">
        <div>
          <h2 id="gateway-routing-title">{{ copy.routingTitle }}</h2>
          <p>{{ copy.routingDescription }}</p>
        </div>
        <dl v-if="providers.length" class="routing-stats">
          <div><dt>{{ t('publicSite.market.providers') }}</dt><dd>{{ n(providers.length) }}</dd></div>
          <div><dt>{{ t('publicSite.market.availableModels') }}</dt><dd>{{ n(modelCount) }}</dd></div>
          <div><dt>{{ t('publicSite.market.groups') }}</dt><dd>{{ n(groupCount) }}</dd></div>
        </dl>
      </div>
      <div v-if="visibleProviders.length" class="route-graph">
        <div class="route-client">
          <div class="route-node"><Laptop :size="26" aria-hidden="true" /></div>
          <span>{{ copy.application }}</span>
        </div>
        <div class="route-center">
          <div class="route-node gateway-core"><Network :size="36" aria-hidden="true" /></div>
          <strong>{{ siteName }}</strong>
        </div>
        <ul class="route-targets">
          <li v-for="provider in visibleProviders" :key="provider.name">
            <RouterLink :to="providerLink(provider.name)">
              <PlatformIcon :platform="provider.name as GroupPlatform" size="lg" aria-hidden="true" />
              <span>{{ platformLabel(provider.name) }}</span>
            </RouterLink>
          </li>
        </ul>
      </div>
      <div class="detail-actions route-actions">
        <RouterLink :to="marketPath" class="detail-button detail-button-dark">
          {{ t('publicSite.market.explore') }} <ArrowRight :size="15" aria-hidden="true" />
        </RouterLink>
        <RouterLink :to="docsPath" class="detail-button">{{ t('publicSite.nav.apiDocs') }}</RouterLink>
      </div>
    </div>
  </section>

  <section id="quick-start" class="gateway-sdk gateway-band" aria-labelledby="gateway-sdk-title">
    <div class="detail-shell">
      <div class="sdk-layout">
        <div class="sdk-intro">
          <h2 id="gateway-sdk-title">{{ copy.sdkTitle }} <span>{{ copy.sdkSubtitle }}</span></h2>
          <p>{{ t('publicHome.quickstart.description') }}</p>
          <ul class="sdk-endpoints" :aria-label="t('publicHome.quickstart.apiEndpoint')">
            <li v-for="endpoint in apiEndpoints" :key="endpoint"><b>POST</b> <code>{{ endpoint }}</code></li>
          </ul>
          <dl class="sdk-summary">
            <div v-for="item in integrationNotes" :key="item.key">
              <div><dt>{{ t(item.title) }}</dt><dd>{{ t(item.description) }}</dd></div>
              <component :is="item.icon" :size="18" aria-hidden="true" />
            </div>
          </dl>
          <div class="detail-actions">
            <RouterLink to="/keys" class="detail-button detail-button-primary">
              {{ t('publicHome.quickstart.createKey.title') }} <ArrowRight :size="15" aria-hidden="true" />
            </RouterLink>
            <RouterLink v-if="showModels" :to="marketPath" class="detail-button">{{ t('publicSite.market.heading') }}</RouterLink>
          </div>
        </div>

        <Tabs id="api-example" v-model="sdkTab" class="sdk-panel">
          <TabsList class="sdk-categories" :aria-label="t('publicHome.examples.protocolsLabel')">
            <TabsTrigger value="api"><CodeXml :size="14" aria-hidden="true" /> API</TabsTrigger>
            <TabsTrigger value="agents"><Terminal :size="14" aria-hidden="true" /> {{ copy.agents }}</TabsTrigger>
            <TabsTrigger value="endpoint"><Link2 :size="14" aria-hidden="true" /> {{ t('publicHome.quickstart.apiEndpoint') }}</TabsTrigger>
          </TabsList>
          <TabsContent value="api" class="sdk-category-panel">
            <Tabs v-model="selectedExample">
              <TabsList class="code-tabs" :aria-label="t('publicHome.examples.protocolsLabel')">
                <TabsTrigger v-for="example in examples" :key="example.id" :value="example.id">
                  {{ example.label }}
                </TabsTrigger>
              </TabsList>
              <TabsContent v-for="example in examples" :key="example.id" :value="example.id">
                <div class="code-frame">
                  <div class="code-file"><span class="file-dots" aria-hidden="true"><i></i><i></i><i></i></span><span>{{ example.file }}</span></div>
                  <pre :aria-label="t('publicHome.examples.requestLabel')" tabindex="0"><code><span v-for="(line, index) in example.code.split('\n')" :key="index" class="source-line"><span class="line-number" aria-hidden="true">{{ index + 1 }}</span><span class="source-text" :class="lineColor(line)">{{ line || ' ' }}</span></span></code></pre>
                </div>
              </TabsContent>
            </Tabs>
          </TabsContent>
          <TabsContent value="agents" class="sdk-category-panel">
            <div class="sdk-instructions">
              <h3>{{ copy.agents }}</h3>
              <p>{{ copy.agentsDescription }}</p>
              <dl>
                <div><dt>{{ t('publicHome.quickstart.apiEndpoint') }}</dt><dd><code>{{ gatewayRoot }}/v1</code></dd></div>
                <div><dt>API Key</dt><dd><code>YOUR_API_KEY</code></dd></div>
                <div><dt>{{ t('publicSite.modelDetail.modelId') }}</dt><dd><code>MODEL_ID</code></dd></div>
              </dl>
            </div>
          </TabsContent>
          <TabsContent value="endpoint" class="sdk-category-panel">
            <div class="sdk-instructions">
              <h3>{{ t('publicHome.quickstart.apiEndpoint') }}</h3>
              <p>{{ t('publicHome.quickstart.connectApp.description') }}</p>
              <dl>
                <div><dt>OpenAI</dt><dd><code>{{ gatewayRoot }}/v1</code></dd></div>
                <div><dt>Claude / Anthropic</dt><dd><code>{{ gatewayRoot }}</code></dd></div>
                <div><dt>Gemini</dt><dd><code>{{ gatewayRoot }}/v1beta</code></dd></div>
              </dl>
            </div>
          </TabsContent>
          <div class="code-actions">
            <button
              type="button"
              class="detail-button compact-button"
              :aria-label="sdkTab === 'api' ? t('publicHome.examples.copyRequest') : t('publicHome.quickstart.copyEndpoint')"
              @click="copyToClipboard(copyValue)"
            >
              <Check v-if="copied" :size="15" aria-hidden="true" /><Copy v-else :size="15" aria-hidden="true" />
              {{ t(copied ? 'publicHome.examples.copied' : 'publicHome.examples.copy') }}
            </button>
            <RouterLink :to="docsPath" class="detail-button compact-button">{{ t('publicSite.nav.apiDocs') }} <ArrowUpRight :size="15" aria-hidden="true" /></RouterLink>
          </div>
        </Tabs>
      </div>

      <div v-if="showModels && providers.length" id="providers" class="sdk-providers">
        <div class="provider-heading">
          <h3>{{ t('publicSite.market.providers') }}</h3>
          <RouterLink :to="marketPath" class="inline-link">{{ t('publicSite.market.explore') }} <ArrowRight :size="15" aria-hidden="true" /></RouterLink>
        </div>
        <ul class="provider-tiles">
          <li v-for="provider in providers" :key="provider.name">
            <RouterLink :to="providerLink(provider.name)" :title="t('publicSite.models.modelCount', { count: provider.count })">
              <PlatformIcon :platform="provider.name as GroupPlatform" size="lg" aria-hidden="true" />
              <span>{{ platformLabel(provider.name) }}</span>
            </RouterLink>
          </li>
        </ul>
      </div>
    </div>
  </section>

  <section v-if="showMonitor" id="reliability" class="gateway-reliability gateway-band" aria-labelledby="gateway-reliability-title">
    <div class="reliability-shell">
      <div class="detail-centered">
        <h2 id="gateway-reliability-title">{{ copy.reliabilityTitle }}</h2>
        <p>{{ copy.reliabilityDescription }}</p>
      </div>
      <div class="reliability-figure">
        <div class="reliability-nodes">
          <span v-for="item in reliabilityItems" :key="item.label">
            <component :is="item.icon" :size="20" aria-hidden="true" />
            {{ item.label }}
          </span>
        </div>
        <div class="figure-divider"><span><GitBranch :size="14" aria-hidden="true" /> {{ copy.failover }}</span></div>
        <RouterLink to="/monitor" class="status-preview">
          <Activity :size="21" aria-hidden="true" />
          {{ t('nav.channelStatus') }}
          <ArrowUpRight :size="15" aria-hidden="true" />
        </RouterLink>
      </div>
      <div class="reliability-cards">
        <article><GitBranch :size="22" aria-hidden="true" /><h3>{{ copy.failover }}</h3><p>{{ copy.failoverDescription }}</p></article>
        <article><Activity :size="22" aria-hidden="true" /><h3>{{ t('nav.channelStatus') }}</h3><p>{{ copy.monitorDescription }}</p></article>
      </div>
    </div>
  </section>

  <section id="faq" class="gateway-faq gateway-band" aria-labelledby="gateway-faq-title">
    <div class="detail-shell faq-layout">
      <div class="faq-intro">
        <h2 id="gateway-faq-title">{{ copy.faqTitle }}</h2>
        <p>{{ copy.faqDescription }}</p>
        <RouterLink :to="localizedPublicPath('/docs', locale)" class="inline-link">{{ t('docs.title') }} <ArrowUpRight :size="15" aria-hidden="true" /></RouterLink>
      </div>
      <div class="faq-rows">
        <details v-for="(item, index) in faqItems" :key="item.question" name="gateway-home-faq" :open="index === 0">
          <summary>{{ item.question }} <ChevronDown :size="18" aria-hidden="true" /></summary>
          <p>{{ item.answer }}</p>
          <RouterLink :to="item.path" class="inline-link">{{ item.link }} <ArrowUpRight :size="14" aria-hidden="true" /></RouterLink>
        </details>
      </div>
    </div>
  </section>

  <section class="gateway-teams gateway-band" aria-labelledby="gateway-teams-title">
    <div class="detail-shell team-shell">
      <div class="detail-centered">
        <h2 id="gateway-teams-title">{{ copy.teamTitle }}<br /><span>{{ copy.teamSubtitle }}</span></h2>
        <p>{{ t('publicHome.services.description') }}</p>
      </div>
      <div class="team-cards">
        <article v-for="item in teamItems" :key="item.key">
          <div class="team-icon"><component :is="item.icon" :size="19" aria-hidden="true" /></div>
          <h3>{{ t(`publicHome.services.${item.key}.title`) }}</h3>
          <p>{{ t(`publicHome.services.${item.key}.description`) }}</p>
        </article>
      </div>
      <div class="detail-actions team-actions">
        <RouterLink to="/keys" class="detail-button detail-button-primary">{{ t('publicHome.quickstart.createKey.title') }} <ArrowRight :size="15" aria-hidden="true" /></RouterLink>
        <RouterLink :to="docsPath" class="detail-button">{{ t('publicSite.nav.apiDocs') }}</RouterLink>
      </div>
    </div>
  </section>

  <section id="get-started" class="gateway-start gateway-band" aria-labelledby="gateway-start-title">
    <div class="detail-shell detail-centered">
      <h2 id="gateway-start-title">{{ copy.startTitle }}<br /><span>{{ copy.startSubtitle }}</span></h2>
      <p>{{ t('publicHome.getStarted.description') }}</p>
      <div class="detail-actions team-actions">
        <RouterLink :to="startPath" class="detail-button detail-button-primary">
          {{ t(isAuthenticated ? 'publicHome.actions.dashboard' : 'publicHome.actions.getStarted') }} <ArrowRight :size="16" aria-hidden="true" />
        </RouterLink>
        <RouterLink :to="localizedPublicPath('/docs/quickstart', locale)" class="detail-button">
          {{ t('publicSite.nav.quickStart') }}
        </RouterLink>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  Activity, ArrowRight, ArrowUpRight, BarChart3, Check, ChevronDown, CircleDollarSign,
  CodeXml, Copy, GitBranch, KeyRound, Laptop, Layers3, Link2, Network, Terminal
} from '@lucide/vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useClipboard } from '@/composables/useClipboard'
import { isLocaleCode, localizedPublicPath } from '@/i18n/localeConfig'
import type { GroupPlatform } from '@/types'
import { platformLabel } from '@/utils/platformColors'
import { gatewayDetailsCopy } from './gatewayDetailsCopy'

const props = defineProps<{
  siteName: string
  providers: readonly { name: string; count: number }[]
  modelCount: number
  groupCount: number
  showModels: boolean
  showMonitor: boolean
  gatewayRoot: string
  startPath: string
  isAuthenticated: boolean
  protocols: readonly { id: string; label: string }[]
  requestExamples: Readonly<Record<string, string>>
}>()

const { t, n, locale } = useI18n()
const { copied, copyToClipboard } = useClipboard()
const copy = computed(() => gatewayDetailsCopy[isLocaleCode(locale.value) ? locale.value : 'en'])
const marketPath = computed(() => localizedPublicPath('/model-plaza', locale.value))
const docsPath = computed(() => localizedPublicPath('/docs/api', locale.value))
const visibleProviders = computed(() => props.providers.slice(0, 5))
const providerLink = (platform: string) => localizedPublicPath(
  `/model-plaza?${new URLSearchParams({ search: platformLabel(platform) })}`, locale.value
)

const apiEndpoints = ['/v1/chat/completions', '/v1/responses', '/v1/messages', '/v1/images/generations']
const integrationNotes = [
  { key: 'api', icon: CodeXml, title: 'publicHome.services.unifiedApi.title', description: 'publicHome.services.unifiedApi.description' },
  { key: 'pricing', icon: CircleDollarSign, title: 'publicHome.services.costs.title', description: 'publicHome.services.costs.description' },
  { key: 'key', icon: KeyRound, title: 'publicHome.services.keys.title', description: 'publicHome.services.keys.description' },
]
const sdkTab = ref('api')
const selectedExample = ref('sdk')
const sdkExample = computed(() => [
  'import OpenAI from "openai";',
  '',
  'const client = new OpenAI({',
  `  baseURL: ${JSON.stringify(props.gatewayRoot + '/v1')},`,
  '  apiKey: process.env.SUB2API_API_KEY,',
  '});',
  '',
  'const response = await client.chat.completions.create({',
  '  model: "MODEL_ID",',
  `  messages: [{ role: "user", content: ${JSON.stringify(t('publicHome.examples.prompt'))} }],`,
  '});',
  '',
  'console.log(response.choices[0]?.message.content);'
].join('\n'))
const examples = computed(() => [
  { id: 'sdk', label: 'OpenAI SDK', file: 'app.ts', code: sdkExample.value },
  ...props.protocols.map(protocol => ({
    ...protocol, file: `${protocol.id}.sh`, code: props.requestExamples[protocol.id] ?? '',
  }))
])
const copyValue = computed(() => sdkTab.value === 'api'
  ? examples.value.find(example => example.id === selectedExample.value)?.code ?? ''
  : `${props.gatewayRoot}/v1`
)

function lineColor(line: string) {
  const text = line.trimStart()
  if (text.startsWith('import') || text.startsWith('const') || text.startsWith('curl')) return 'code-keyword'
  if (text.includes('"')) return 'code-string'
  return ''
}

const reliabilityItems = computed(() => [
  { icon: CodeXml, label: copy.value.application },
  { icon: GitBranch, label: copy.value.channelRouting },
  { icon: Layers3, label: t('publicSite.market.availableModels') }
])
const teamItems = [
  { key: 'keys', icon: KeyRound }, { key: 'modelSelection', icon: Layers3 },
  { key: 'costs', icon: CircleDollarSign }, { key: 'usage', icon: BarChart3 }
] as const
const faqItems = computed(() => [
  {
    question: copy.value.faqStart, answer: copy.value.answerStart,
    path: localizedPublicPath('/home#quick-start', locale.value), link: t('publicSite.nav.quickStart')
  },
  {
    question: copy.value.faqModels, answer: copy.value.answerModels,
    path: props.showModels ? marketPath.value : docsPath.value,
    link: t(props.showModels ? 'publicSite.market.heading' : 'publicSite.nav.apiDocs')
  },
  {
    question: copy.value.faqCosts, answer: copy.value.answerCosts,
    path: props.showModels ? marketPath.value : '/usage',
    link: t(props.showModels ? 'publicSite.market.heading' : 'publicHome.services.usage.title')
  },
  {
    question: copy.value.faqStatus, answer: copy.value.answerStatus,
    path: props.showMonitor ? '/monitor' : docsPath.value,
    link: t(props.showMonitor ? 'nav.channelStatus' : 'publicSite.nav.apiDocs')
  }
])
</script>

<style scoped>
.gateway-band {
  color: var(--gateway-ink);
  scroll-margin-top: 96px;
}
.gateway-band :is(a, button, summary, pre):focus-visible {
  outline: 2px solid var(--gateway-accent);
  outline-offset: 4px;
}
.detail-shell {
  width: min(calc(100% - 2 * var(--gateway-gutter, 32px)), var(--gateway-width, 1232px));
  margin-inline: auto;
}
.gateway-band h2 {
  font-family: var(--gateway-display, inherit);
  font-size: clamp(30px, 4vw, 48px);
  font-weight: 700;
  letter-spacing: -.045em;
  line-height: 1.1;
  text-wrap: balance;
}
.gateway-band h2 span { color: var(--gateway-muted); }
.routing-heading p, .sdk-intro > p, .detail-centered > p, .faq-intro > p {
  max-width: 590px;
  margin-top: 18px;
  color: var(--gateway-muted);
  font-size: 16px;
  line-height: 1.65;
}
.detail-centered { text-align: center; }
.detail-centered > p { margin-inline: auto; }
.detail-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.detail-button {
  display: inline-flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  max-width: 100%;
  padding: 9px 16px;
  border: 1px solid var(--gateway-line);
  border-radius: 8px;
  background: var(--gateway-surface);
  color: var(--gateway-ink);
  font-size: 14px;
  font-weight: 600;
  line-height: 1.3;
  transition: background .16s ease, box-shadow .16s ease, color .16s ease;
}
.detail-button:hover { background: var(--gateway-soft); }
.detail-button svg { flex-shrink: 0; }
.detail-button-dark {
  border-color: var(--gateway-ink);
  background: var(--gateway-ink);
  color: var(--gateway-surface);
}
.detail-button-dark:hover { background: color-mix(in srgb, var(--gateway-ink) 88%, var(--gateway-surface)); }
.detail-button-primary {
  border-color: transparent;
  border-radius: 999px;
  background: #5355df;
  color: white;
  box-shadow: 0 6px 18px rgb(83 85 223 / 25%);
  padding-inline: 20px;
}
.detail-button-primary:hover { background: #4345cc; box-shadow: 0 8px 22px rgb(83 85 223 / 34%); }
.inline-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--gateway-ink);
  font-size: 14px;
  font-weight: 600;
  text-decoration: underline;
  text-decoration-color: transparent;
  text-underline-offset: 4px;
  transition: text-decoration-color .16s ease;
}
.inline-link:hover { text-decoration-color: currentColor; }

.gateway-routing {
  background: linear-gradient(var(--gateway-surface), var(--gateway-soft) 60%, var(--gateway-surface));
  padding-block: clamp(76px, 11vw, 144px);
}
.routing-heading { display: flex; align-items: end; justify-content: space-between; gap: 26px; }
.routing-stats { display: flex; flex-wrap: wrap; gap: 14px clamp(18px, 3vw, 44px); }
.routing-stats > div { display: flex; flex-direction: column-reverse; }
.routing-stats dt { color: var(--gateway-muted); font-size: 13px; }
.routing-stats dd { font-size: clamp(26px, 3.6vw, 38px); font-weight: 700; font-variant-numeric: tabular-nums; line-height: 1.2; }
.route-graph {
  position: relative;
  isolation: isolate;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  align-items: center;
  justify-items: center;
  max-width: 896px;
  min-height: 430px;
  margin: 12px auto;
  padding: 42px 20px;
  background: radial-gradient(ellipse at center, rgb(92 94 233 / 7%), transparent 59%);
}
.route-client, .route-center {
  display: grid;
  gap: 12px;
  justify-items: center;
  position: relative;
  min-width: 0;
  max-width: 100%;
  color: var(--gateway-muted);
  font-size: 13px;
}
.route-client::after, .route-center::after {
  position: absolute;
  z-index: -1;
  top: 50%;
  left: 75%;
  width: min(31vw, 275px);
  height: 1px;
  background: linear-gradient(90deg, var(--gateway-line), var(--gateway-accent), var(--gateway-line));
  content: '';
}
.route-node {
  display: grid;
  width: 72px;
  height: 72px;
  place-items: center;
  color: var(--gateway-ink);
  border: 1px solid var(--gateway-line);
  border-radius: 18px;
  background: var(--gateway-surface);
  box-shadow: 0 2px 8px rgb(0 0 0 / 5%);
}
.gateway-core { width: 106px; height: 106px; border-radius: 26px; color: var(--gateway-accent); box-shadow: 0 0 0 9px rgb(87 86 232 / 7%), 0 7px 18px rgb(87 86 232 / 14%); }
.route-center strong { color: var(--gateway-ink); font-weight: 650; overflow-wrap: anywhere; }
.route-targets { position: relative; display: grid; min-width: 0; max-width: 100%; gap: 14px; justify-self: end; }
.route-targets::before {
  position: absolute;
  z-index: -1;
  left: -23px;
  top: 25px;
  bottom: 25px;
  border-inline-start: 1px solid var(--gateway-line);
  content: '';
}
.route-targets li { position: relative; min-width: 0; }
.route-targets li::before {
  position: absolute;
  z-index: -1;
  width: 23px;
  right: 100%;
  top: 50%;
  border-block-start: 1px solid var(--gateway-line);
  content: '';
}
.route-targets a {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border: 1px solid var(--gateway-line);
  border-radius: 10px;
  background: var(--gateway-surface);
  font-size: 14px;
  font-weight: 600;
}
.route-targets a:hover { color: var(--gateway-accent); }
.route-targets span { overflow-wrap: anywhere; }
.route-actions { justify-content: center; margin-top: 22px; }

.gateway-sdk { padding-block: clamp(74px, 10vw, 128px); }
.sdk-layout { display: grid; align-items: start; gap: clamp(24px, 4vw, 56px); grid-template-columns: minmax(0, 5fr) minmax(0, 7fr); }
.sdk-intro { min-width: 0; }
.sdk-endpoints {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-block: 20px 26px;
}
.sdk-endpoints li {
  padding: 6px 8px;
  border: 1px solid var(--gateway-line);
  border-radius: 6px;
  background: var(--gateway-soft);
  font-size: 11px;
  overflow-wrap: anywhere;
}
.sdk-endpoints b { color: var(--gateway-accent); font-family: ui-monospace, monospace; font-weight: 650; }
.sdk-summary { margin-bottom: 24px; border: 1px solid var(--gateway-line); border-radius: 12px; }
.sdk-summary > div { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 13px 16px; }
.sdk-summary > div + div { border-top: 1px solid var(--gateway-line); }
.sdk-summary :is(dd, svg) { color: var(--gateway-muted); }
.sdk-summary dt { font-size: 14px; font-weight: 650; }
.sdk-summary dd { margin-top: 4px; font-size: 12px; line-height: 1.5; }
.sdk-summary svg { flex-shrink: 0; }
.sdk-panel { min-width: 0; scroll-margin-top: 96px; }
.sdk-panel :deep([role="tabpanel"]) { min-width: 0; margin: 0; }
.sdk-categories, .code-tabs { width: 100%; height: auto; justify-content: start; gap: 4px; overflow-x: auto; border: 1px solid var(--gateway-line); }
.sdk-categories { border-radius: 11px; padding: 4px; background: var(--gateway-soft); }
.sdk-categories :deep(button) { display: inline-flex; gap: 7px; min-height: 34px; max-width: none; flex-shrink: 0; color: var(--gateway-muted); font-weight: 600; }
.sdk-categories :deep(button[data-state="active"]) { color: var(--gateway-ink); background: var(--gateway-surface); }
.sdk-panel :deep(.sdk-category-panel) { margin-top: 12px; }
.code-tabs { border-radius: 10px 10px 0 0; border-bottom-width: 0; padding: 6px; background: var(--gateway-surface); }
.code-tabs :deep(button) { min-height: 31px; max-width: none; flex-shrink: 0; padding-inline: 10px; font-size: 12px; color: var(--gateway-muted); }
.code-tabs :deep(button[data-state="active"]) { color: var(--gateway-ink); background: var(--gateway-soft); box-shadow: none; }
.code-frame { border: 1px solid var(--gateway-line); border-radius: 0 0 10px 10px; overflow: hidden; background: var(--gateway-soft); }
.code-file { position: relative; display: flex; align-items: center; justify-content: center; gap: 12px; min-height: 38px; padding-inline: 70px; border-bottom: 1px solid var(--gateway-line); color: var(--gateway-muted); font: 12px ui-monospace, monospace; }
.file-dots { position: absolute; left: 14px; display: flex; gap: 6px; }
.file-dots i { display: inline-block; width: 9px; height: 9px; border-radius: 50%; background: #ff6059; }
.file-dots i:nth-child(2) { background: #f9bd2e; }
.file-dots i:nth-child(3) { background: #28c743; }
.code-frame pre {
  min-height: 302px;
  overflow: auto;
  padding-block: 14px;
  padding-inline: 12px 16px;
  color: var(--gateway-ink);
  font: 13px/1.7 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  tab-size: 2;
}
.code-frame pre:focus-visible { outline-offset: -2px; }
.source-line { display: flex; min-width: max-content; gap: 13px; }
.line-number { width: 22px; color: var(--gateway-muted); text-align: end; user-select: none; opacity: .8; }
.code-keyword { color: var(--gateway-accent); }
.code-string { color: #08734f; }
:global(.dark) .code-string { color: #80dab8; }
.code-actions { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 8px; margin-top: 10px; }
.compact-button { min-height: 36px; font-size: 12px; }
.sdk-instructions { min-height: 360px; padding: 24px; border: 1px solid var(--gateway-line); border-radius: 10px; background: var(--gateway-soft); }
.sdk-instructions h3 { font-size: 18px; font-weight: 650; }
.sdk-instructions p { margin-top: 7px; color: var(--gateway-muted); font-size: 14px; line-height: 1.65; }
.sdk-instructions dl { display: grid; gap: 10px; margin-top: 18px; }
.sdk-instructions dl > div { display: grid; gap: 4px; padding: 10px 12px; border: 1px solid var(--gateway-line); border-radius: 7px; background: var(--gateway-surface); }
.sdk-instructions dt { color: var(--gateway-muted); font-size: 12px; }
.sdk-instructions dd { overflow-wrap: anywhere; font-size: 13px; }
.sdk-providers { margin-top: clamp(42px, 7vw, 82px); padding-top: 32px; border-top: 1px solid var(--gateway-line); }
.provider-heading { display: flex; flex-wrap: wrap; align-items: baseline; justify-content: space-between; gap: 10px; margin-bottom: 18px; }
.provider-heading h3 { color: var(--gateway-muted); font-size: 15px; font-weight: 600; }
.provider-tiles { display: grid; grid-template-columns: repeat(auto-fill, minmax(155px, 1fr)); gap: 10px; }
.provider-tiles a { display: flex; min-height: 70px; align-items: center; justify-content: center; gap: 8px; padding: 10px; border: 1px solid var(--gateway-line); border-radius: 10px; background: var(--gateway-soft); font-size: 13px; font-weight: 600; }
.provider-tiles a:hover { color: var(--gateway-accent); border-color: var(--gateway-accent); }
.provider-tiles span { min-width: 0; overflow-wrap: anywhere; }

.gateway-reliability { padding-block: clamp(66px, 10vw, 112px); }
.reliability-shell { width: min(100% - 64px, 1024px); margin-inline: auto; }
.reliability-figure { margin-top: 45px; padding: clamp(18px, 3vw, 32px); border: 1px solid var(--gateway-line); border-radius: 16px; background: var(--gateway-surface); }
.reliability-nodes { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.reliability-nodes > span { display: flex; min-width: 0; flex-direction: column; justify-content: center; align-items: center; min-height: 84px; gap: 8px; padding: 10px; text-align: center; border: 1px solid var(--gateway-line); border-radius: 10px; background: var(--gateway-soft); font-size: 13px; font-weight: 600; }
.reliability-nodes svg { color: var(--gateway-accent); }
.figure-divider { position: relative; margin-block: 27px; border-top: 1px dashed var(--gateway-line); }
.figure-divider span { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); display: inline-flex; gap: 7px; align-items: center; padding: 1px 10px; background: var(--gateway-surface); color: var(--gateway-muted); font-size: 12px; white-space: nowrap; }
.status-preview { display: flex; align-items: center; justify-content: center; min-height: 52px; gap: 10px; border: 1px solid rgb(34 197 94 / 26%); border-radius: 10px; background: rgb(34 197 94 / 7%); color: #08734f; font-size: 14px; font-weight: 650; }
:global(.dark) .status-preview { color: #80dab8; }
.status-preview:hover { background: rgb(34 197 94 / 12%); }
.reliability-cards { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; margin-top: 26px; }
.reliability-cards article { padding: 23px; border: 1px solid var(--gateway-line); border-radius: 12px; background: var(--gateway-soft); }
.reliability-cards svg { color: var(--gateway-accent); }
.reliability-cards h3 { margin-top: 9px; font-size: 16px; font-weight: 650; }
.reliability-cards p { margin-top: 6px; color: var(--gateway-muted); font-size: 14px; line-height: 1.6; }

.gateway-faq { padding-block: clamp(72px, 11vw, 128px); }
.faq-layout { display: grid; grid-template-columns: minmax(0, 2fr) minmax(0, 3fr); align-items: start; gap: clamp(26px, 6vw, 72px); }
.faq-intro .inline-link { margin-top: 20px; }
.faq-rows details { padding-block: 18px; border-bottom: 1px solid var(--gateway-line); }
.faq-rows details:first-child { padding-top: 0; }
.faq-rows details:last-child { border-bottom: 0; }
.faq-rows summary { display: flex; align-items: center; justify-content: space-between; gap: 14px; padding-block: 7px; font-size: 18px; font-weight: 600; cursor: pointer; list-style: none; }
.faq-rows summary::-webkit-details-marker { display: none; }
.faq-rows summary svg { color: var(--gateway-muted); flex-shrink: 0; transition: transform .18s ease; }
.faq-rows details[open] summary svg { transform: rotate(180deg); }
.faq-rows details p { margin: 7px 0; color: var(--gateway-muted); font-size: 15px; line-height: 1.65; }
.faq-rows details .inline-link { margin-top: 3px; }

.gateway-teams {
  padding-block: clamp(74px, 10vw, 128px);
  background: radial-gradient(ellipse at center top, rgb(87 86 232 / 8%), transparent 58%);
}
.team-shell { max-width: 1152px; }
.team-cards { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 16px; margin-block: 44px 30px; }
.team-cards article { padding: 22px; border: 1px solid var(--gateway-line); border-radius: 12px; background: var(--gateway-surface); }
.team-icon { display: grid; width: 38px; height: 38px; place-items: center; margin-bottom: 13px; border-radius: 8px; background: rgb(87 86 232 / 10%); color: var(--gateway-accent); }
.team-cards h3 { font-size: 16px; font-weight: 650; }
.team-cards p { margin-top: 6px; color: var(--gateway-muted); font-size: 14px; line-height: 1.55; }
.team-actions { justify-content: center; }
.gateway-start {
  padding-block: clamp(82px, 12vw, 160px);
  border-block: 1px solid var(--gateway-line);
  background: radial-gradient(ellipse at center bottom, rgb(87 86 232 / 12%), transparent 70%);
}
.gateway-start h2 { font-size: clamp(38px, 5vw, 60px); }
.gateway-start .detail-actions { margin-top: 26px; }

@media (max-width: 940px) {
  .sdk-layout { grid-template-columns: minmax(0, 1fr); }
  .routing-heading { align-items: start; flex-direction: column; }
  .team-cards { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 640px) {
  .reliability-shell { width: calc(100% - 32px); }
  .route-graph { min-height: 335px; padding: 32px 0; }
  .gateway-core { width: 70px; height: 70px; border-radius: 16px; }
  .gateway-core svg { width: 28px; height: 28px; }
  .route-node:not(.gateway-core) { width: 52px; height: 52px; border-radius: 12px; }
  .route-client, .route-center { gap: 8px; font-size: 11px; text-align: center; }
  .route-client::after, .route-center::after { width: 19vw; }
  .route-targets { gap: 9px; justify-self: stretch; }
  .route-targets::before { left: -8px; top: 16px; bottom: 16px; }
  .route-targets li::before { width: 8px; }
  .route-targets a { gap: 6px; padding: 8px 6px; border-radius: 8px; font-size: 11px; }
  .route-targets a svg { width: 14px; height: 14px; }
  .provider-tiles { grid-template-columns: repeat(auto-fill, minmax(116px, 1fr)); }
  .faq-layout, .reliability-cards { grid-template-columns: minmax(0, 1fr); }
  .team-cards { gap: 12px; }
  .team-cards article { padding: 14px; }
}
@media (max-width: 365px) {
  .team-cards { grid-template-columns: minmax(0, 1fr); }
}
@media (prefers-reduced-motion: reduce) {
  .gateway-band * { transition: none !important; }
}
</style>
