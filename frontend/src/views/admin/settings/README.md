# Management settings

Settings has nine sections on the same protected route record. The section
registry is shared by the tab strip, the old-link redirects, and the command
palette. Update that registry when adding or renaming a section.

SettingsView owns the form, the save dock, and the provider, confirmation,
and step-up dialogs. The panels stay mounted across section switches and
inject one typed settings context, so changing sections preserves edits.
The general save still submits the whole settings draft. Cards with independent
save or test actions keep their original requests. The unsaved badge tracks the
shared settings and web search drafts in memory.

During an upstream merge, bring template changes to the matching panel. Apply
state, validators and operation changes to useSettingsForm, and update
useSettingsPayload and its parity fixtures when the update request changes.
