# Shared interface components

Sub2API uses **shadcn-vue** source components with **Reka UI**, **Lucide Vue**, existing Chart.js visualizations, and the application's business controls.

The initial Button, Badge, Card, Input, Tabs, Switch, Separator, Dialog and Skeleton sources were retrieved from the official `new-york-v4` registry on 2026-10-25. The MIT notice is retained in [LICENSE](./LICENSE). Sources: [registry](https://www.shadcn-vue.com/r/styles/new-york-v4/button.json), [documentation](https://www.shadcn-vue.com/docs), [upstream](https://github.com/unovue/shadcn-vue).

This application intentionally retains Tailwind 3.4. The imported components use `@/lib/utils` (`clsx` + Tailwind 3-compatible `tailwind-merge`), `tailwindcss-animate`, and compatible shadow/ring/outline utilities. Theme tokens are declared in `src/style.css` and mapped in `tailwind.config.js`. Numeric `primary-50` to `primary-950` colors remain available for existing business pages. Card has no default padding/flex layout, so page composition controls its spacing.

Use `Button as-child` around router links to preserve navigation semantics. Set `type="button"` for non-submit controls inside forms. Use `Input` for simple fields and existing searchable business selectors for model/account/group data. `common/UiDialog.vue` adapts shadcn Dialog to controlled `show`/`close` state; it blocks outside dismissal by default and accepts `closeOnEscape=false` during saving. Legacy BaseDialog retains its existing scroll/stack contract and shares the same visual tokens.

Do not replace specialized pricing, billing, CAPTCHA, permissions, virtualized tables or asynchronous selection logic as part of a styling change. Maintain light/dark, narrow screens, loading/error/empty states, keyboard focus and translated labels. No external font or image service is required by this theme.

The admin redesign adds Select, Combobox, Table, Sheet, Dropdown Menu, Tooltip, Popover and Pagination, adapted from the same registry for Tailwind 3. Use the appropriate provider for Tooltip, and pair Sheet with its Title and Description. Use `flex-1 min-h-0 overflow-y-auto` for the scrollable body of a Sheet with persistent header and footer.

- **DataTable** wraps TanStack Vue Table. Supply `data`, `columns`, and a stable `getRowId` for selectable or refreshed rows; bind named models for controlled sorting, pagination, selection, visibility, or density. Passing `rowCount` enables server pagination; add `manualSorting` when the server also sorts. Callers keep ownership of requests, loading, empty state and row actions. Business tables with existing pagination are not automatically replaced.
- **Command** accepts groups of actions and emits a selected entry, with a controlled `CommandDialog` available for overlays. Application shortcuts and navigation belong in the caller.
- **Form** uses the native Vue submit event and browser constraints with the accessible `FormField` wrapper. Bind its `controlAttrs` to the input or trigger, and pass existing validation errors as text. No validation rules, save methods or additional schema dependencies are required.
- **Toast** provides a themed `Toaster` and `toast` facade from Vue Sonner. Mount it once where you adopt it; existing application notifications retain their renderer until migration.

Run `corepack pnpm codemod:colors` after merging older class pairs from upstream. The optional `--check` flag reports remaining deterministic replacements without modifying files. Keep unsupported colors and intentional variants unless their role is clear.
