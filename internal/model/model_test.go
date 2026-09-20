package model

import (
	"encoding/json"
	"testing"
)

func TestTaskPatchOptionalSemantics(t *testing.T) {
	var p TaskPatch
	if err := json.Unmarshal([]byte(`{"title":"Latte","notes":null,"is_completed":true}`), &p); err != nil {
		t.Fatal(err)
	}
	if !p.Title.Set || p.Title.Value != "Latte" {
		t.Fatalf("title not decoded: %+v", p.Title)
	}
	if !p.Notes.Set || p.Notes.Value != nil {
		t.Fatalf("explicit null must be Set with nil value: %+v", p.Notes)
	}
	if !p.IsCompleted.Set || !p.IsCompleted.Value {
		t.Fatalf("is_completed not decoded")
	}
	if p.DepartmentID.Set || p.DueDate.Set || p.AssigneeID.Set || p.ItemType.Set {
		t.Fatalf("absent fields must not be Set: %+v", p)
	}
	if p.Empty() {
		t.Fatal("patch with fields must not be empty")
	}

	var only json.RawMessage = []byte(`{"updated_at":"2026-01-01T00:00:00Z"}`)
	var q TaskPatch
	if err := json.Unmarshal(only, &q); err != nil {
		t.Fatal(err)
	}
	if !q.Empty() || q.UpdatedAt == nil {
		t.Fatalf("timestamp-only patch should be empty but carry updated_at: %+v", q)
	}
}
