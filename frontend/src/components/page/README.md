# Page building blocks

Use `PageHeader` for a route's title, description and actions. Use `Toolbar`
for control groups and `FilterBar` where one form owns a search action.
The caller owns filter values, loading, and when to refetch. Place auxiliary
actions in the `actions` slot and mark non-submit buttons as `type="button"`.

Use `StatCard` for metrics returned by the existing API. Format values, units,
and trends in the caller and use the `chart` slot for an existing chart.
`StatusBadge` pairs a translated label with a semantic tone so that color
is not the only status cue. `EmptyState` can be used in the empty slot of a
table or within a section. `SectionCard` provides labeled header, body and
footer regions; set `flush` when the body already handles its own padding.

These components are shared across admin, user and public routes. Keep route
guards, API calls, form validation and permission checks in the consuming views.
