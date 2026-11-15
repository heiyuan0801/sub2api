import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import { h, ref } from 'vue'
import type { ColumnDef, PaginationState, RowSelectionState } from '@tanstack/vue-table'
import en from '@/i18n/locales/en/design'
import DataTable from '../DataTable.vue'

const rows = [
  { id: '2', name: 'Zeta', platform: 'Gemini' },
  { id: '1', name: 'Alpha', platform: 'OpenAI' },
  { id: '3', name: 'Beta', platform: 'Claude' }
]
type Row = typeof rows[number]
const columns: ColumnDef<Row>[] = [
  { accessorKey: 'name', header: 'Name' },
  { accessorKey: 'platform', header: 'Platform' }
]

function renderTable(pageSize = 2) {
  const selected = ref<RowSelectionState>({})
  const page = ref<PaginationState>({ pageIndex: 0, pageSize })
  const wrapper = mount({
    render: () => h(DataTable<Row>, {
      data: rows,
      columns,
      getRowId: row => row.id,
      selectable: true,
      pagination: page.value,
      'onUpdate:pagination': next => { page.value = next },
      rowSelection: selected.value,
      'onUpdate:rowSelection': next => { selected.value = next }
    })
  }, {
    attachTo: document.body,
    global: { plugins: [createI18n({ legacy: false, locale: 'en', messages: { en } })] }
  })
  return { wrapper, selected }
}

describe('DataTable', () => {
  it('sorts before paging and keeps selection keyed by row ID across pages', async () => {
    const { wrapper, selected } = renderTable()
    try {
      const sort = wrapper.get('th button')
      expect(wrapper.findAll('tbody td:nth-child(2)').map(cell => cell.text())).toEqual(['Zeta', 'Alpha'])
      await sort.trigger('click')
      expect(wrapper.get('th[aria-sort]').attributes('aria-sort')).toBe('ascending')
      expect(wrapper.findAll('tbody td:nth-child(2)').map(cell => cell.text())).toEqual(['Alpha', 'Beta'])
      await wrapper.get('thead input').setValue(true)
      expect(selected.value).toEqual({ 1: true, 3: true })
      await wrapper.get('[aria-label="Next"]').trigger('click')
      expect(wrapper.findAll('tbody td:nth-child(2)').map(cell => cell.text())).toEqual(['Zeta'])
      expect((wrapper.get('thead input').element as HTMLInputElement).checked).toBe(false)
      expect(selected.value).toEqual({ 1: true, 3: true })
    } finally {
      wrapper.unmount()
    }
  })

  it('changes row density and hides the requested column', async () => {
    const { wrapper } = renderTable()
    try {
      const density = wrapper.get('button[aria-pressed]')
      await density.trigger('click')
      expect(density.attributes('aria-pressed')).toBe('true')
      expect(wrapper.get('table').attributes('data-density')).toBe('compact')

      const trigger = wrapper.get('button[aria-haspopup="menu"]')
      await trigger.trigger('keydown', { key: 'Enter' })
      await flushPromises()
      const menuItems = [...document.querySelectorAll<HTMLElement>('[role="menuitemcheckbox"]')]
      const platform = menuItems.find(el => el.textContent?.trim() === 'Platform')
      expect(platform?.getAttribute('aria-checked')).toBe('true')
      platform?.click()
      await flushPromises()
      expect(platform?.getAttribute('aria-checked')).toBe('false')
      expect(wrapper.get('thead').text()).not.toContain('Platform')
      expect(wrapper.get('tbody').text()).not.toContain('Gemini')
    } finally {
      wrapper.unmount()
    }
  })
})
