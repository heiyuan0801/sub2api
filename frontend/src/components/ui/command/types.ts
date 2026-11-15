import type { Component } from 'vue'

export interface CommandEntry {
  id: string
  label: string
  description?: string
  keywords?: string[]
  disabled?: boolean
  icon?: Component
}

export interface CommandGroup {
  id: string
  label?: string
  items: readonly CommandEntry[]
}
