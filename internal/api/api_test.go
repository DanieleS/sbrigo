package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/danieles/sbrigo/internal/auth"
	"github.com/danieles/sbrigo/internal/db"
	"github.com/danieles/sbrigo/internal/model"
	"github.com/danieles/sbrigo/internal/realtime"
	"github.com/danieles/sbrigo/internal/store"
)

// newTestServer boots the API against the database in SBRIGO_TEST_DATABASE_URL, migrating a
// fresh schema. The test is skipped when the variable is unset.
func newTestServer(t *testing.T) (*httptest.Server, *store.Store, *realtime.Broker) {
	t.Helper()
	url := os.Getenv("SBRIGO_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("SBRIGO_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := db.Migrate(ctx, pool, log); err != nil {
		t.Fatal(err)
	}

	st := store.New(pool)
	broker := realtime.NewBroker(log)
	authn := auth.NewAuthenticator(auth.NewSignedSessions(auth.NewSigner("0123456789abcdef0123456789abcdef")), "agent-key", log)
	mux := http.NewServeMux()
	New(st, broker, authn, log).Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, st, broker
}

func do(t *testing.T, srv *httptest.Server, method, path string, body any, out any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, srv.URL+path, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(auth.APIKeyHeader, "agent-key")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, raw, err)
		}
	}
	return resp
}

func TestUnauthenticatedIsRejected(t *testing.T) {
	srv, _, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/api/v1/tasks")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestGroceryLifecycle(t *testing.T) {
	srv, _, broker := newTestServer(t)
	events, cancel := broker.Subscribe()
	defer cancel()

	var deps []model.Department
	if resp := do(t, srv, http.MethodGet, "/api/v1/departments", nil, &deps); resp.StatusCode != 200 || len(deps) == 0 {
		t.Fatalf("seeded departments expected, status %d len %d", resp.StatusCode, len(deps))
	}

	// Create with a client-generated id (offline replay scenario).
	id := uuid.New()
	created := time.Now().Add(-time.Minute).UTC().Truncate(time.Microsecond)
	var task model.Task
	resp := do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{
		"id": id, "title": "  Latte ", "department_id": deps[0].ID, "created_at": created, "updated_at": created,
	}, &task)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status %d", resp.StatusCode)
	}
	if task.ID != id || task.Title != "Latte" || task.ItemType != model.ItemTypeGrocery || !task.UpdatedAt.Equal(created) {
		t.Fatalf("unexpected task %+v", task)
	}
	expectEvent(t, events, "task.created")

	// Replaying the same creation is idempotent.
	var again model.Task
	if resp := do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{"id": id, "title": "Latte"}, &again); resp.StatusCode != http.StatusOK || again.ID != id {
		t.Fatalf("replay status %d id %s", resp.StatusCode, again.ID)
	}

	// Tick it with a client timestamp.
	tick := created.Add(30 * time.Second)
	if resp := do(t, srv, http.MethodPatch, "/api/v1/tasks/"+id.String(), map[string]any{"is_completed": true, "updated_at": tick}, &task); resp.StatusCode != 200 || !task.IsCompleted {
		t.Fatalf("patch status %d task %+v", resp.StatusCode, task)
	}
	expectEvent(t, events, "task.updated")

	// A stale offline edit (older timestamp) loses: 409 with the current row.
	var current model.Task
	if resp := do(t, srv, http.MethodPatch, "/api/v1/tasks/"+id.String(), map[string]any{"title": "Latte scremato", "updated_at": created.Add(10 * time.Second)}, &current); resp.StatusCode != http.StatusConflict {
		t.Fatalf("stale patch status %d", resp.StatusCode)
	}
	if current.Title != "Latte" || !current.IsCompleted {
		t.Fatalf("conflict response must carry the stored row: %+v", current)
	}

	// Filtering.
	var open []model.Task
	do(t, srv, http.MethodGet, "/api/v1/tasks?type=grocery&completed=false", nil, &open)
	if len(open) != 0 {
		t.Fatalf("expected no open items, got %d", len(open))
	}
	var since []model.Task
	do(t, srv, http.MethodGet, "/api/v1/tasks?since="+created.Format(time.RFC3339Nano), nil, &since)
	if len(since) != 1 {
		t.Fatalf("since filter: got %d", len(since))
	}

	// Validation errors.
	if resp := do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{"title": ""}, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty title status %d", resp.StatusCode)
	}
	if resp := do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{"title": "x", "department_id": uuid.New()}, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown department status %d", resp.StatusCode)
	}
	if resp := do(t, srv, http.MethodPatch, "/api/v1/tasks/"+id.String(), map[string]any{"updated_at": tick}, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty patch status %d", resp.StatusCode)
	}
	if resp := do(t, srv, http.MethodPatch, "/api/v1/tasks/"+uuid.New().String(), map[string]any{"title": "x"}, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing task status %d", resp.StatusCode)
	}

	// Bulk removal of completed items.
	var bulk struct {
		Deleted int         `json:"deleted"`
		IDs     []uuid.UUID `json:"ids"`
	}
	if resp := do(t, srv, http.MethodDelete, "/api/v1/tasks?completed=true&type=grocery", nil, &bulk); resp.StatusCode != 200 || bulk.Deleted != 1 {
		t.Fatalf("bulk delete status %d body %+v", resp.StatusCode, bulk)
	}
	expectEvent(t, events, "task.deleted")
	if resp := do(t, srv, http.MethodGet, "/api/v1/tasks/"+id.String(), nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("deleted task still readable: %d", resp.StatusCode)
	}
}

func TestGeneralTaskWithAssignee(t *testing.T) {
	srv, st, _ := newTestServer(t)
	user, err := st.UpsertUser(context.Background(), "logto-sub-1", "anna@example.com", "Anna")
	if err != nil {
		t.Fatal(err)
	}
	// Second login refreshes the profile instead of duplicating it.
	same, err := st.UpsertUser(context.Background(), "logto-sub-1", "anna@example.com", "Anna Rossi")
	if err != nil || same.ID != user.ID || same.DisplayName != "Anna Rossi" {
		t.Fatalf("upsert user: %v %+v", err, same)
	}

	due := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	var task model.Task
	resp := do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{
		"title": "Pagare bollo auto", "item_type": "general_task", "assignee_id": user.ID, "due_date": due, "notes": "Scade a fine mese",
	}, &task)
	if resp.StatusCode != http.StatusCreated || task.AssigneeID == nil || *task.AssigneeID != user.ID || task.DueDate == nil || !task.DueDate.Equal(due) {
		t.Fatalf("status %d task %+v", resp.StatusCode, task)
	}

	// Reassign to nobody and clear the notes with explicit nulls.
	if resp := do(t, srv, http.MethodPatch, "/api/v1/tasks/"+task.ID.String(), map[string]any{"assignee_id": nil, "notes": nil}, &task); resp.StatusCode != 200 {
		t.Fatalf("patch status %d", resp.StatusCode)
	}
	if task.AssigneeID != nil || task.Notes != nil {
		t.Fatalf("nulls not applied: %+v", task)
	}

	var users []model.User
	do(t, srv, http.MethodGet, "/api/v1/users", nil, &users)
	if len(users) != 1 {
		t.Fatalf("users: %d", len(users))
	}
}

func TestSupermarketOrder(t *testing.T) {
	srv, _, _ := newTestServer(t)
	var deps []model.Department
	do(t, srv, http.MethodGet, "/api/v1/departments", nil, &deps)

	var m model.Supermarket
	if resp := do(t, srv, http.MethodPost, "/api/v1/supermarkets", map[string]any{"name": "Esselunga"}, &m); resp.StatusCode != http.StatusCreated {
		t.Fatalf("create supermarket %d", resp.StatusCode)
	}
	order := []uuid.UUID{deps[2].ID, deps[0].ID, deps[1].ID}
	var got []model.DepartmentOrder
	if resp := do(t, srv, http.MethodPut, fmt.Sprintf("/api/v1/supermarkets/%s/department-order", m.ID), map[string]any{"department_ids": order}, &got); resp.StatusCode != 200 {
		t.Fatalf("put order %d", resp.StatusCode)
	}
	if len(got) != 3 || got[0].DepartmentID != deps[2].ID || got[0].SortOrder != 1 || got[2].SortOrder != 3 {
		t.Fatalf("order %+v", got)
	}
	if resp := do(t, srv, http.MethodPut, fmt.Sprintf("/api/v1/supermarkets/%s/department-order", m.ID), map[string]any{"department_ids": []uuid.UUID{deps[0].ID, deps[0].ID}}, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("duplicate ids accepted: %d", resp.StatusCode)
	}
	if resp := do(t, srv, http.MethodPut, fmt.Sprintf("/api/v1/supermarkets/%s/department-order", uuid.New()), map[string]any{"department_ids": order}, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown supermarket: %d", resp.StatusCode)
	}

	// Deleting a department drops it from the order and detaches tasks.
	var task model.Task
	do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{"title": "Pane", "department_id": deps[2].ID}, &task)
	if resp := do(t, srv, http.MethodDelete, "/api/v1/departments/"+deps[2].ID.String(), nil, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete department %d", resp.StatusCode)
	}
	do(t, srv, http.MethodGet, fmt.Sprintf("/api/v1/supermarkets/%s/department-order", m.ID), nil, &got)
	if len(got) != 2 {
		t.Fatalf("order after delete %+v", got)
	}
	do(t, srv, http.MethodGet, "/api/v1/tasks/"+task.ID.String(), nil, &task)
	if task.DepartmentID != nil {
		t.Fatalf("task still references deleted department")
	}
	if resp := do(t, srv, http.MethodDelete, "/api/v1/supermarkets/"+m.ID.String(), nil, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete supermarket %d", resp.StatusCode)
	}
}

func expectEvent(t *testing.T, ch <-chan realtime.Event, typ string) {
	t.Helper()
	select {
	case ev := <-ch:
		if ev.Type != typ {
			t.Fatalf("event %q, want %q", ev.Type, typ)
		}
	case <-time.After(time.Second):
		t.Fatalf("no %q event", typ)
	}
}

func TestTaskPositionOrdering(t *testing.T) {
	srv, _, _ := newTestServer(t)
	var a, b, c model.Task
	do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{"title": "A"}, &a)
	do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{"title": "B"}, &b)
	do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{"title": "C", "position": 1}, &c)
	if a.Position == 0 || b.Position <= a.Position || c.Position != 1 {
		t.Fatalf("positions: a=%v b=%v c=%v", a.Position, b.Position, c.Position)
	}
	var list []model.Task
	do(t, srv, http.MethodGet, "/api/v1/tasks", nil, &list)
	if list[0].Title != "C" || list[1].Title != "A" || list[2].Title != "B" {
		t.Fatalf("order by position: %s %s %s", list[0].Title, list[1].Title, list[2].Title)
	}
	// Drag A after B: midpoint between B and +infinity is B+1000 on the client; any larger value works.
	do(t, srv, http.MethodPatch, "/api/v1/tasks/"+a.ID.String(), map[string]any{"position": b.Position + 1000}, &a)
	do(t, srv, http.MethodGet, "/api/v1/tasks", nil, &list)
	if list[2].Title != "A" {
		t.Fatalf("reorder: %s %s %s", list[0].Title, list[1].Title, list[2].Title)
	}
}

func TestLists(t *testing.T) {
	srv, _, _ := newTestServer(t)

	var lists []model.ListSummary
	do(t, srv, http.MethodGet, "/api/v1/lists", nil, &lists)
	if len(lists) != 2 {
		t.Fatalf("seeded lists expected, got %d", len(lists))
	}
	var groceryDefault, tasksDefault model.ListSummary
	for _, l := range lists {
		if l.Kind == model.ItemTypeGrocery {
			groceryDefault = l
		} else {
			tasksDefault = l
		}
	}

	// A task without list_id lands in the default list of its type.
	var task model.Task
	do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{"title": "Latte"}, &task)
	if task.ListID != groceryDefault.ID || task.ItemType != model.ItemTypeGrocery {
		t.Fatalf("default list not applied: %+v", task)
	}

	// A second tasks list; creating into it sets item_type from the list kind.
	var farmacia model.List
	if resp := do(t, srv, http.MethodPost, "/api/v1/lists", map[string]any{"name": "Farmacia", "kind": "grocery", "sort_order": 20}, &farmacia); resp.StatusCode != http.StatusCreated {
		t.Fatalf("create list %d", resp.StatusCode)
	}
	if resp := do(t, srv, http.MethodPost, "/api/v1/lists", map[string]any{"name": "x", "kind": "nope"}, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid kind accepted: %d", resp.StatusCode)
	}
	due := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	var t2 model.Task
	do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{"title": "Aspirina", "list_id": farmacia.ID, "item_type": "general_task", "due_date": due}, &t2)
	if t2.ListID != farmacia.ID || t2.ItemType != model.ItemTypeGrocery {
		t.Fatalf("list kind must win over item_type: %+v", t2)
	}
	if resp := do(t, srv, http.MethodPost, "/api/v1/tasks", map[string]any{"title": "x", "list_id": uuid.New()}, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown list accepted: %d", resp.StatusCode)
	}

	// Moving a task to a list of another kind changes its type.
	do(t, srv, http.MethodPatch, "/api/v1/tasks/"+task.ID.String(), map[string]any{"list_id": tasksDefault.ID}, &task)
	if task.ListID != tasksDefault.ID || task.ItemType != model.ItemTypeGeneralTask {
		t.Fatalf("move between kinds: %+v", task)
	}

	// Filtering and summaries.
	var inFarmacia []model.Task
	do(t, srv, http.MethodGet, "/api/v1/tasks?list_id="+farmacia.ID.String(), nil, &inFarmacia)
	if len(inFarmacia) != 1 || inFarmacia[0].Title != "Aspirina" {
		t.Fatalf("list filter: %+v", inFarmacia)
	}
	do(t, srv, http.MethodGet, "/api/v1/lists", nil, &lists)
	for _, l := range lists {
		if l.ID == farmacia.ID && (l.OpenCount != 1 || l.NextDue == nil || !l.NextDue.Equal(due)) {
			t.Fatalf("summary: %+v", l)
		}
	}

	// Deleting the only tasks list is refused; deleting a sibling grocery list cascades.
	if resp := do(t, srv, http.MethodDelete, "/api/v1/lists/"+tasksDefault.ID.String(), nil, nil); resp.StatusCode != http.StatusConflict {
		t.Fatalf("last list delete status %d", resp.StatusCode)
	}
	if resp := do(t, srv, http.MethodDelete, "/api/v1/lists/"+farmacia.ID.String(), nil, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete list status %d", resp.StatusCode)
	}
	if resp := do(t, srv, http.MethodGet, "/api/v1/tasks/"+t2.ID.String(), nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("task of deleted list still present: %d", resp.StatusCode)
	}
}
