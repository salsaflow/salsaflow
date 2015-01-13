package sprintly

import (
	"time"
)

// Product represents a Sprintly product.
type Product struct {
	Id        int        `json:"id,omitempty"`
	Name      string     `json:"name,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	Webhook   string     `json:"webhook,omitempty"`
	Archived  bool       `json:"archived,omitempty"`
	Admin     bool       `json:"admin,omitempty"`
	Email     *Email     `json:"email,omitempty"`
}

// Email represents the email section of a Product.
type Email struct {
	Tests   string `json:"tests,omitempty"`
	Tasks   string `json:"tasks,omitempty"`
	Stories string `json:"stories,omitempty"`
	Defects string `json:"defects,omitempty"`
	Backlog string `json:"backlog,omitempty"`
}
