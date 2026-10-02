package models

// SidebarCounts holds unread-message counts for user-visible addresses.
type SidebarCounts struct {
	Unread    int         `json:"unread"`
	Addresses map[int]int `json:"addresses"`
}
