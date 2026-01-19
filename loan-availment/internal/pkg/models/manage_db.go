package models

type ManageDbRequest struct {
	Operation      string `json:"operation" binding:"required"`
	CollectionName string `json:"collection_name" binding:"required"`
	Body           any    `json:"body,omitempty"`
	Filter         any    `json:"filter,omitempty"`
}
