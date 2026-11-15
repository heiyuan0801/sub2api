# Account management UI

The account list uses the TanStack adapter on desktop. Small screens retain
the existing card layout and row actions. Requests, selected and filtered
bulk targets, and fetching and refresh behavior live in AccountsView.

The three existing form controllers opt into AccountFormSheet from the list.
Their default centered presentation remains for legacy callers and tests.
The sheet keeps the full form mounted: edit section links scroll to the fields
without hiding required inputs or interrupting authorization flows. Legacy
select menus, tooltips and confirmation dialogs rendered inside the sheet
use its overlay target so they stay inside the modal focus boundary.

See [platform-config](./platform-config/README.md) for the credential and
scheduling fields already driven by shared metadata. Other fields continue
to use their existing controllers and validators. The edit summary only
uses stats or upstream snapshots already loaded by the page.
