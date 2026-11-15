package domain

// GroupDisplayTag is a user-facing badge for a group.
type GroupDisplayTag struct {
	Label string `json:"label"`
	Color string `json:"color"`
	Icon  string `json:"icon"`
}
