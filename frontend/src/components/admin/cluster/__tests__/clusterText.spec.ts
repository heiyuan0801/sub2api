import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import zh from '@/i18n/locales/zh/cluster'
import en from '@/i18n/locales/en/cluster'
import { createClusterText } from '../clusterText'

let locale: 'zh' | 'en' = 'zh'
const text = createClusterText((key, params) => {
  const message = key.split('.').reduce((value: any, part) => value?.[part], { zh, en }[locale])
  if (typeof message !== 'string') throw new Error(`Missing translation: ${key}`)
  return message.replace(/\{(\w+)\}/g, (_, name: string) => String(params?.[name] ?? `{${name}}`))
})
const here = dirname(fileURLToPath(import.meta.url))
const original = '这条记录暂无对应的中文说明，请展开原始日志查看详情。'

describe('cluster log explanations', () => {
  it('translates every static alert and preflight detail currently emitted by the backend', () => {
    // Contract coverage: a new English backend message must get a readable label.
    const observer = readFileSync(resolve(here, '../../../../../../backend/internal/clusterops/observe.go'), 'utf8')
    const preflight = readFileSync(resolve(here, '../../../../../../backend/internal/clusterops/preflight.go'), 'utf8')
    const messages = [
      ...Array.from(observer.matchAll(/add\(n, "[^"]+", "[^"]+", "([^"]+)"\)/g), m => m[1]!),
      ...Array.from(observer.matchAll(/Message: "([^"]+)"/g), m => m[1]!),
      ...Array.from(preflight.matchAll(/add\("[^"]+", "[^"]+", "([^"]+)"\)/g), m => m[1]!)
    ]
    expect(messages.length).toBeGreaterThan(25)
    for (const message of messages) {
      expect(text.message(message), message).not.toBe(original)
      expect(text.message(message), message).toMatch(/[\u3400-\u9fff]/)
    }
  })

  it('explains historical startup roles without changing node identifiers', () => {
    expect(text.event({ kind: 'node_started', message: 'node started with role primary' })).toBe('节点已启动，当前角色：主节点。')
    expect(text.event({ kind: 'node_started', message: 'node started with role worker' })).toBe('节点已启动，当前角色：工作节点。')
    expect(text.label('redisRoles', 'master')).toBe('主库')
    expect(text.label('requests', 'finished')).toBe('请求已结束')
  })

  it('preserves numbers and the administrator reason, including multiline text', () => {
    expect(text.event({ kind: 'control_changed', message: 'maintenance=false capacity=128 weight=30: 部署: v2\n保留业务' })).toBe('维护排空：已关闭；最大并发：128；分配权重：30。操作原因：部署: v2\n保留业务')
    expect(text.event({ kind: 'control_changed', message: 'maintenance=true capacity=128 weight=30: <script>alert(1)</script>' })).toContain('<script>alert(1)</script>')
  })

  it('distinguishes an alert being raised from its historical resolution', () => {
    expect(text.event({ kind: 'alert_raised', message: 'expected one application primary; observed 0' })).toBe('集群应有 1 个应用主节点，当前检测到 0 个。')
    expect(text.event({ kind: 'alert_resolved', message: 'cluster:primary' })).toBe('“整个集群”的“主节点数量异常”告警已解除。')
    expect(text.event({ kind: 'alert_resolved', message: 'region:hk:offline' })).toBe('“region:hk”的“节点心跳超时”告警已解除。')
  })

  it('recovers the node scope from historical resolved-alert keys', () => {
    expect(text.scope('', 'region:hk:offline')).toBe('region:hk')
    expect(text.scope('', 'cluster:primary')).toBe('整个集群')
    expect(text.scope('us', 'hk:offline')).toBe('us')
  })

  it('explains manual verification without claiming it was passed', () => {
    expect(text.label('statuses', 'manual')).toBe('需人工验证')
    expect(text.message('WAL archiving is configured; verify successful delivery and restore separately')).toContain('能否恢复仍需单独验证')
    expect(text.label('statuses', 'unknown')).toBe('无法确认')
  })

  it('explains billing operations while retaining IDs and no-debit semantics', () => {
    expect(text.event({ kind: 'billing_review', message: '019d2f1a-0440-7000-8000-000000000001: review recorded' })).toContain('不会自动补扣款或退款')
    expect(text.event({ kind: 'balance_cache_invalidated', message: 'user 42; rebuild from database on next use' })).toBe('已清除用户 42 的余额缓存，下次使用时将从数据库重新加载。')
  })

  it('keeps unknown or malformed messages available for inspection, without guessing meaning', () => {
    for (const message of ['new backend failure', '__proto__', 'constructor']) expect(text.message(message)).toBe(original)
    expect(text.event({ kind: 'control_changed', message: 'maintenance=maybe capacity=?' })).toBe(original)
    expect(text.label('statuses', 'future_status')).toBe('未知类型（future_status）')
    expect(text.label('__proto__', 'x')).toBe('未知类型（x）')
  })

  it('follows the selected locale and resolves every declared label', () => {
    for (const group of ['roles', 'redisRoles', 'kinds', 'alerts', 'checks', 'statuses', 'requests', 'severities'] as const) {
      for (const [value, label] of Object.entries(zh.cluster.log[group])) expect(text.label(group, value)).toBe(label)
    }
    locale = 'en'
    expect(text.event({ kind: 'node_started', message: 'node started with role worker' })).toBe('Node started with role: Worker.')
    locale = 'zh'
  })
})
