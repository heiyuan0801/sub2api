<template>
  <div v-if="hasData" class="performance-plot">
    <Line :data="chartData" :options="chartOptions" :plugins="chartPlugins" :aria-label="metricLabel" />
  </div>
  <div v-else class="performance-plot performance-plot-empty" role="status">
    <span class="performance-plot-message">
      <ChartNoAxesCombined :size="20" aria-hidden="true" />
      {{ t(loading ? 'publicSite.performance.loading' : 'publicSite.performance.noData') }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChartNoAxesCombined } from '@lucide/vue'
import { useMutationObserver } from '@vueuse/core'
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
  type ChartData,
  type ChartOptions,
  type Plugin,
  type PointStyle,
  type ScriptableContext,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import type { PlazaPerformancePoint, PlazaPerformanceRange } from '@/api/modelPlaza'
import { formatMonitorRate } from '@/features/channel-monitor-v2/monitorFormat'

ChartJS.register(CategoryScale, LinearScale, LineElement, PointElement, Filler, Tooltip)

const props = defineProps<{
  trend: PlazaPerformancePoint[]
  metric: 'ttft' | 'availability'
  range: PlazaPerformanceRange
  loading: boolean
}>()

const { t, locale } = useI18n()
const isTtft = computed(() => props.metric === 'ttft')
const themeRoot = typeof document === 'undefined' ? null : document.documentElement
const isDark = ref(themeRoot?.classList.contains('dark') ?? false)
useMutationObserver(themeRoot, () => {
  isDark.value = themeRoot?.classList.contains('dark') ?? false
}, { attributeFilter: ['class'] })

const metricLabel = computed(() => t(isTtft.value ? 'publicSite.performance.averageTtft' : 'publicSite.performance.successRate'))
const values = computed(() => props.trend.map(bucket => {
  if (isTtft.value) return bucket.avg_ttft_ms
  return bucket.success_rate == null ? null : bucket.success_rate * 100
}))
const hasData = computed(() => values.value.some(value => value != null))
// Inspired by DesignBookmark Stat Card; chart geometry always follows live samples.
const colors = computed(() => isDark.value ? {
  line: isTtft.value ? '#73a9ff' : '#34d17a',
  fill: isTtft.value ? 'rgba(115,169,255,.20)' : 'rgba(52,209,122,.20)',
  warning: '#ffc25c',
  critical: '#ff8090',
  surface: '#262626',
  panel: '#262626',
  text: '#9a9a9a',
  ink: '#fff',
  grid: '#3b3b3b',
  guide: '#828282',
} : {
  line: isTtft.value ? '#4b7ee8' : '#168252',
  fill: isTtft.value ? 'rgba(75,126,232,.15)' : 'rgba(22,130,82,.15)',
  warning: '#bc6c10',
  critical: '#d13651',
  surface: '#fff',
  panel: '#f8f8f7',
  text: '#747474',
  ink: '#171717',
  grid: '#e8e8e7',
  guide: '#a0a0a0',
})

const chartPlugins: Plugin<'line'>[] = [{
  id: 'plaza-performance-guide',
  beforeDatasetsDraw(chart) {
    const x = chart.tooltip?.getActiveElements()[0]?.element.x
    if (x == null) return
    const { ctx, chartArea } = chart
    ctx.save()
    ctx.strokeStyle = colors.value.guide
    ctx.setLineDash([3, 3])
    ctx.beginPath()
    ctx.moveTo(x, chartArea.top)
    ctx.lineTo(x, chartArea.bottom)
    ctx.stroke()
    ctx.restore()
  },
}]

function abnormal(bucket: PlazaPerformancePoint) {
  return bucket.abnormal ?? (bucket.failures ?? 0) > 0
}

function pointMarker(bucket: PlazaPerformancePoint): { color: string; style: PointStyle; radius: number } {
  const palette = colors.value
  if (isTtft.value && bucket.ttft_status === 'warning') {
    return { color: palette.warning, style: 'triangle', radius: 5.5 }
  }
  if (isTtft.value ? bucket.ttft_status === 'critical' : abnormal(bucket)) {
    return { color: palette.critical, style: 'rectRot', radius: 5 }
  }
  return { color: palette.line, style: 'circle', radius: 2 }
}

const chartData = computed<ChartData<'line', (number | null)[]>>(() => {
  const tickDate = new Intl.DateTimeFormat(locale.value, props.range === '24h'
    ? { hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }
    : { month: 'numeric', day: 'numeric', ...(props.range === '7d' ? { hour: '2-digit', hourCycle: 'h23' } : {}) })
  const markers = props.trend.map(pointMarker)
  return {
    labels: props.trend.map(bucket => tickDate.format(new Date(bucket.bucket_start))),
    datasets: [{
      label: metricLabel.value,
      data: values.value,
      borderColor: colors.value.line,
      borderWidth: 2,
      fill: 'origin',
      backgroundColor(context: ScriptableContext<'line'>) {
        const { ctx, chartArea } = context.chart
        if (!chartArea) return 'transparent'
        const gradient = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom)
        gradient.addColorStop(0, colors.value.fill)
        gradient.addColorStop(1, 'transparent')
        return gradient
      },
      clip: { top: 6, right: 0, bottom: 0, left: 0 },
      pointBackgroundColor: markers.map(point => point.color),
      pointStyle: markers.map(point => point.style),
      pointBorderColor: colors.value.panel,
      pointBorderWidth: 1.5,
      pointRadius: markers.map(point => point.radius),
      pointHoverRadius: 6,
      pointHitRadius: 10,
      tension: 0,
      spanGaps: false,
    }],
  }
})

const chartOptions = computed<ChartOptions<'line'>>(() => {
  const { text, grid, surface, ink } = colors.value
  const fullDate = new Intl.DateTimeFormat(locale.value, {
    month: 'long', day: 'numeric', hour: '2-digit', minute: '2-digit', hourCycle: 'h23',
  })
  const percentage = new Intl.NumberFormat(locale.value, {
    style: 'percent', minimumFractionDigits: 2, maximumFractionDigits: 2,
  })
  const tickNumber = new Intl.NumberFormat(locale.value, { maximumFractionDigits: 1 })
  const validValues = values.value.filter((value): value is number => value != null)
  const minimum = validValues.length ? Math.min(...validValues) : 0
  const tickFont = { size: 11, family: 'ui-sans-serif, system-ui, sans-serif' }
  return {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,
    interaction: { mode: 'index', axis: 'x', intersect: false },
    layout: { padding: { right: 8, top: 10 } },
    scales: {
      x: {
        grid: { display: false },
        border: { color: grid },
        ticks: {
          color: text,
          font: tickFont,
          padding: 6,
          maxRotation: 0,
          autoSkip: true,
          autoSkipPadding: 12,
          maxTicksLimit: 5,
        },
        offset: true,
      },
      y: {
        min: isTtft.value ? 0 : Math.max(0, Math.floor((minimum - 5) / 10) * 10),
        max: isTtft.value ? undefined : 100,
        grid: { color: grid, drawTicks: false },
        border: { display: false, dash: [3, 4] },
        ticks: {
          color: text,
          font: tickFont,
          maxTicksLimit: 4,
          padding: 8,
          callback: value => isTtft.value ? `${tickNumber.format(Number(value))} ms` : `${tickNumber.format(Number(value))}%`,
        },
      },
    },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: surface,
        titleColor: ink,
        bodyColor: text,
        borderColor: grid,
        borderWidth: 1,
        cornerRadius: 12,
        padding: 10,
        usePointStyle: true,
        boxPadding: 3,
        callbacks: {
          labelPointStyle(item) {
            return { pointStyle: pointMarker(props.trend[item.dataIndex]).style, rotation: 0 }
          },
          title(items) {
            const bucket = props.trend[items[0]?.dataIndex]
            return bucket ? fullDate.format(new Date(bucket.bucket_start)) : ''
          },
          label(item) {
            const value = item.parsed.y
            if (value == null) return t('publicSite.performance.noData')
            const formatted = isTtft.value
              ? `${formatMonitorRate(value, locale.value)} ms`
              : percentage.format(value / 100)
            return `${metricLabel.value} · ${formatted}`
          },
          afterLabel(item) {
            const bucket = props.trend[item.dataIndex]
            if (!bucket) return ''
            if (isTtft.value) {
              return bucket.ttft_status === 'critical' || bucket.ttft_status === 'warning'
                ? t('publicSite.performance.abnormal')
                : ''
            }
            if (bucket.failures != null) return t('publicSite.performance.failures', { count: bucket.failures })
            return abnormal(bucket) ? t('publicSite.performance.abnormal') : ''
          },
        },
      },
    },
  }
})
</script>

<style scoped>
.performance-plot {
  position: relative;
  min-width: 0;
  height: clamp(12.5rem, 21vw, 15rem);
  margin-top: .5rem;
}
.performance-plot :deep(canvas) { touch-action: pan-y pinch-zoom; }
.performance-plot-empty {
  display: grid;
  place-items: center;
  color: var(--detail-muted, #747474);
}
.performance-plot-empty::before {
  position: absolute;
  inset: .75rem 0 .75rem;
  background: repeating-linear-gradient(to bottom, transparent 0, transparent calc(25% - 1px), var(--detail-border, #ececec) calc(25% - 1px), var(--detail-border, #ececec) 25%);
  mask-image: linear-gradient(90deg, transparent, black 12%, black 88%, transparent);
  content: '';
}
.performance-plot-message {
  position: relative;
  display: grid;
  justify-items: center;
  gap: .55rem;
  padding: .7rem 1rem;
  background: var(--detail-subtle, #f8f8f7);
  border-radius: 12px;
  font-size: .8125rem;
}
</style>
