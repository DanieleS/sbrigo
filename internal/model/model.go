// Package model holds the domain types shared by the store, the API and the realtime layer.
package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Item types accepted in tasks.item_type.
const (
	ItemTypeGrocery     = "grocery"
	ItemTypeGeneralTask = "general_task"
)

// ValidItemType reports whether t is one of the supported item types.
func ValidItemType(t string) bool {
	return t == ItemTypeGrocery || t == ItemTypeGeneralTask
}

// User is a family member synchronised from the identity provider.
type User struct {
	ID          uuid.UUID `json:"id"`
	IdentityID  string    `json:"identity_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// Department groups grocery items and carries the legend shown in the UI.
type Department struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	DefaultSortOrder int       `json:"default_sort_order"`
}

// Supermarket is a shop with its own aisle order.
type Supermarket struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// DepartmentOrder is one entry of the aisle sequence of a supermarket.
type DepartmentOrder struct {
	DepartmentID uuid.UUID `json:"department_id"`
	SortOrder    int       `json:"sort_order"`
}

// List groups tasks. Its kind decides whether it holds grocery items or general tasks.
type List struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListSummary is a list with the preview figures shown on its card.
type ListSummary struct {
	List
	OpenCount    int        `json:"open_count"`
	DoneCount    int        `json:"done_count"`
	OverdueCount int        `json:"overdue_count"`
	NextDue      *time.Time `json:"next_due"`
}

// Task is either a grocery item or a general household task.
type Task struct {
	ID           uuid.UUID  `json:"id"`
	ListID       uuid.UUID  `json:"list_id"`
	Title        string     `json:"title"`
	Notes        *string    `json:"notes"`
	IsCompleted  bool       `json:"is_completed"`
	ItemType     string     `json:"item_type"`
	DepartmentID *uuid.UUID `json:"department_id"`
	AssigneeID   *uuid.UUID `json:"assignee_id"`
	DueDate      *time.Time `json:"due_date"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Optional distinguishes "field absent" from "field explicitly set (possibly to null)" in JSON patches.
type Optional[T any] struct {
	Set   bool
	Value T
}

// UnmarshalJSON marks the field as set and decodes the value (null decodes to the zero value).
func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		var zero T
		o.Value = zero
		return nil
	}
	return json.Unmarshal(b, &o.Value)
}

// TaskPatch is a partial update. UpdatedAt is the client-side timestamp of the change and drives
// last-write-wins conflict resolution; when nil the server clock is used.
type TaskPatch struct {
	ListID       Optional[uuid.UUID]  `json:"list_id"`
	Title        Optional[string]     `json:"title"`
	Notes        Optional[*string]    `json:"notes"`
	IsCompleted  Optional[bool]       `json:"is_completed"`
	ItemType     Optional[string]     `json:"item_type"`
	DepartmentID Optional[*uuid.UUID] `json:"department_id"`
	AssigneeID   Optional[*uuid.UUID] `json:"assignee_id"`
	DueDate      Optional[*time.Time] `json:"due_date"`
	UpdatedAt    *time.Time           `json:"updated_at"`
}

// Empty reports whether the patch changes nothing.
func (p TaskPatch) Empty() bool {
	return !p.ListID.Set && !p.Title.Set && !p.Notes.Set && !p.IsCompleted.Set && !p.ItemType.Set &&
		!p.DepartmentID.Set && !p.AssigneeID.Set && !p.DueDate.Set
}

// TaskFilter narrows a task listing.
type TaskFilter struct {
	ListID       *uuid.UUID
	ItemType     string
	IsCompleted  *bool
	DepartmentID *uuid.UUID
	AssigneeID   *uuid.UUID
	UpdatedSince *time.Time
}
