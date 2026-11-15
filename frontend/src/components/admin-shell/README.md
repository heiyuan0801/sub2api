# Console shell

The existing sidebar is the source of menu labels, feature switches, simple mode
and custom links. This directory assigns real admin routes to presentation groups
and renders the console controls and mobile drawer. The command palette consumes
the same visible menu and resolves its entries through the existing router, whose
guards still enforce access. Redirects are classified with their final pages.

Settings section commands use the shared Settings tab registry and the
`section` query. The view handles the URL without resetting the settings
form. Desktop sidebar width and theme preferences are saved locally.
