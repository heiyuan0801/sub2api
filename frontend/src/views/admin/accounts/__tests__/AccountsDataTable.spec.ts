import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import en from '@/i18n/locales/en'
import type { AccountListItem } from '@/types'
import AccountsDataTable from '../AccountsDataTable.vue'

const columns = [
  { key: 'name', label: 'Account', sortable: true },
  { key: 'priority', label: 'Priority', sortable: true }
]

function accountRows(count: number): AccountListItem[] {
  return Array.from({ length: count }, (_, index) => ({
    id: index + 1,
    name: `Account ${index + 1}`,
    priority: count - index
  } as AccountListItem))
}

function mountDesktopTable(rows: AccountListItem[]) {
  return mount(AccountsDataTable, {
    props: {
      columns, data: rows, desktop: true, density: 'compact', loading: false,
      selectedIds: [2], sortKey: 'name', sortOrder: 'asc'
    },
    global: { plugins: [createI18n({ legacy: false, locale: 'en', messages: { en } })] }
  })
}

describe('AccountsDataTable', () => {
  it('renders the whole supplied server page and identifies selections by account ID', async () => {
    const rows = accountRows(30)
    const wrapper = mountDesktopTable(rows)
    try {
      expect(wrapper.findAll('tbody tr[data-row-id]')).toHaveLength(30)
      expect(wrapper.get('tr[data-state="selected"]').attributes('data-row-id')).toBe('2')
      await wrapper.setProps({ data: [...rows].reverse() })
      expect(wrapper.get('tbody tr[data-row-id]').attributes('data-row-id')).toBe('30')
      expect(wrapper.get('tr[data-state="selected"]').attributes('data-row-id')).toBe('2')
    } finally {
      wrapper.unmount()
    }
  })

  it('emits the existing ascending-first server-sort contract without reordering rows locally', async () => {
    const wrapper = mountDesktopTable(accountRows(3))
    try {
      await wrapper.get('th[data-column-id="priority"] button').trigger('click')
      expect(wrapper.emitted('sort')).toEqual([['priority', 'asc']])
      expect(wrapper.get('tbody tr[data-row-id]').attributes('data-row-id')).toBe('1')

      await wrapper.setProps({ sortKey: 'priority', sortOrder: 'asc' })
      await wrapper.get('th[data-column-id="priority"] button').trigger('click')
      expect(wrapper.emitted('sort')).toEqual([['priority', 'asc'], ['priority', 'desc']])
    } finally {
      wrapper.unmount()
    }
  })
})
