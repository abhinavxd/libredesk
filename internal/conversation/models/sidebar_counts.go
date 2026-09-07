package models

// SidebarCounts holds open-conversation counts for inbox sidebar badges.
type SidebarCounts struct {
	Assigned   int         `json:"assigned"`
	Mentioned  int         `json:"mentioned"`
	Unassigned int         `json:"unassigned"`
	All        int         `json:"all"`
	Views      map[int]int `json:"views"`
}
