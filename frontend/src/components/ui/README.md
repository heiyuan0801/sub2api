# Shared interface components

Sub2API uses **shadcn-vue** source components with **Reka UI**, **Lucide Vue**, existing Chart.js visualizations, and the application's business controls.

The initial Button, Badge, Card, Input, Tabs, Switch, Separator, Dialog and Skeleton sources were retrieved from the official `new-york-v4` registry on 2026-10-25. The MIT notice is retained in [LICENSE](./LICENSE). Sources: [registry](https://www.shadcn-vue.com/r/styles/new-york-v4/button.json), [documentation](https://www.shadcn-vue.com/docs), [upstream](https://github.com/unovue/shadcn-vue).

This application intentionally retains Tailwind 3.4. The imported components use `@/lib/utils` (`clsx` + Tailwind 3-compatible `tailwind-merge`), `tailwindcss-animate`, and compatible shadow/ring/outline utilities. Theme tokens are declared in `src/style.css` and mapped in `tailwind.config.js`. Numeric `primary-50` to `primary-950` colors remain available for existing business pages. Card has no default padding/flex layout, so page composition controls its spacing.

Use `Button as-child` around router links to preserve navigation semantics. Set `type="button"` for non-submit controls inside forms. Use `Input` for simple fields and existing searchable business selectors for model/account/group data. `common/UiDialog.vue` adapts shadcn Dialog to controlled `show`/`close` state; it blocks outside dismissal by default and accepts `closeOnEscape=false` during saving. Legacy BaseDialog retains its existing scroll/stack contract and shares the same visual tokens.

Do not replace specialized pricing, billing, CAPTCHA, permissions, virtualized tables or asynchronous selection logic as part of a styling change. Maintain light/dark, narrow screens, loading/error/empty states, keyboard focus and translated labels. No external font or image service is required by this theme.
