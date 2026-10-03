package org

import (
	"testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	svc := NewService(NewState(), t.TempDir())
	mustOK(t, svc.InitRoot("root", "HQ"))
	mustOK(t, svc.AddDepartment("eng", "Engineering", "root"))
	mustOK(t, svc.AddDepartment("backend", "Backend", "eng"))
	mustOK(t, svc.AddPost("p-dev", "Developer", "backend"))
	mustOK(t, svc.AddPost("p-lead", "Lead", "eng"))
	mustOK(t, svc.AddEmployee("alice", "Alice"))
	mustOK(t, svc.AddEmployee("bob", "Bob"))
	return svc
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func wantErr(t *testing.T, err error, contains string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", contains)
	}
	if !containsStr(err.Error(), contains) {
		t.Fatalf("error %q does not contain %q", err.Error(), contains)
	}
}

func containsStr(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestAdjacentAssignmentsAllowed(t *testing.T) {
	svc := newTestService(t)
	a1, err := svc.AddAssignment("alice", "p-dev", "2026-01-01", "2026-03-01")
	mustOK(t, err)
	if a1 != "a1" {
		t.Fatalf("first assignment id = %q, want a1", a1)
	}
	// Exactly touching intervals are allowed.
	_, err = svc.AddAssignment("alice", "p-lead", "2026-03-01", "2026-06-01")
	mustOK(t, err)
	// Open-ended follow-up starting at the previous end is allowed too.
	_, err = svc.AddAssignment("alice", "p-lead", "2026-06-01", "")
	mustOK(t, err)

	// Same post can be held by multiple employees concurrently.
	_, err = svc.AddAssignment("bob", "p-dev", "2026-01-01", "")
	mustOK(t, err)
}

func TestOverlapRejected(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.AddAssignment("alice", "p-dev", "2026-01-01", "")
	mustOK(t, err)
	_, err = svc.AddAssignment("alice", "p-lead", "2026-12-31", "2027-01-02")
	wantErr(t, err, "overlaps")

	_, err = svc.AddAssignment("alice", "p-dev", "2026-01-01", "2026-01-01")
	wantErr(t, err, "later than start")
	_, err = svc.AddAssignment("alice", "p-dev", "2026-02-30", "")
	wantErr(t, err, "invalid date")
	_, err = svc.AddAssignment("ghost", "p-dev", "2026-01-01", "")
	wantErr(t, err, "not found")
	_, err = svc.AddAssignment("alice", "ghost", "2026-01-01", "")
	wantErr(t, err, "not found")
}

func TestDepartmentTreeRules(t *testing.T) {
	svc := newTestService(t)
	wantErr(t, svc.MoveDepartment("root", "eng"), "cannot be moved")
	wantErr(t, svc.MoveDepartment("eng", "eng"), "its own parent")
	wantErr(t, svc.MoveDepartment("eng", "backend"), "descendant")
	wantErr(t, svc.MoveDepartment("eng", "nope"), "not found")

	// Moving backend directly under root is valid.
	mustOK(t, svc.MoveDepartment("backend", "root"))
}

func TestTransferSucceedsAndInheritsEnd(t *testing.T) {
	svc := newTestService(t)
	id, err := svc.AddAssignment("alice", "p-dev", "2026-01-01", "2026-06-01")
	mustOK(t, err)
	newID, err := svc.Transfer(id, "p-lead", "2026-03-01")
	mustOK(t, err)

	old := svc.State().Assignments[id]
	next := svc.State().Assignments[newID]
	if old.End != "2026-03-01" {
		t.Fatalf("old end = %q, want 2026-03-01", old.End)
	}
	if next.Start != "2026-03-01" || next.End != "2026-06-01" {
		t.Fatalf("new interval = [%s, %s), want [2026-03-01, 2026-06-01)", next.Start, next.End)
	}

	hist, err := svc.History("alice")
	mustOK(t, err)
	if len(hist) != 2 {
		t.Fatalf("history length = %d, want 2", len(hist))
	}
}

func TestTransferFailureKeepsOriginalIntact(t *testing.T) {
	svc := newTestService(t)
	id, err := svc.AddAssignment("alice", "p-dev", "2026-01-01", "2026-06-01")
	mustOK(t, err)

	// Every failure path must leave the original interval untouched and must
	// not create the new assignment.
	checkUntouched := func() {
		t.Helper()
		old := svc.State().Assignments[id]
		if old.End != "2026-06-01" {
			t.Fatalf("original end changed to %q after failed transfer", old.End)
		}
		if len(svc.State().Assignments) != 1 {
			t.Fatalf("assignment count = %d, want 1 (no partial transfer)", len(svc.State().Assignments))
		}
	}

	_, err = svc.Transfer(id, "ghost", "2026-03-01")
	wantErr(t, err, "not found")
	checkUntouched()

	// Boundary dates are rejected too.
	_, err = svc.Transfer(id, "p-lead", "2026-01-01")
	wantErr(t, err, "strictly after")
	checkUntouched()
	_, err = svc.Transfer(id, "p-lead", "2026-06-01")
	wantErr(t, err, "strictly before")
	checkUntouched()
	_, err = svc.Transfer("missing", "p-lead", "2026-03-01")
	wantErr(t, err, "not found")
	checkUntouched()
}

func TestQueryDateAndPathUsesCurrentTree(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.AddAssignment("alice", "p-dev", "2026-01-01", "")
	mustOK(t, err)

	views, err := svc.QueryDate("2026-02-01")
	mustOK(t, err)
	if len(views) != 2 {
		t.Fatalf("views = %d, want 2 (unassigned employee kept)", len(views))
	}
	av := views[0].Assignment
	if av == nil {
		t.Fatal("alice should be assigned on 2026-02-01")
	}
	if got := joinStrings(av.Path); got != "root/eng/backend" {
		t.Fatalf("path = %q, want root/eng/backend", got)
	}
	if views[1].Assignment != nil {
		t.Fatal("bob should have empty assignment")
	}

	// Move backend under root: historical assignment is resolved via new tree.
	mustOK(t, svc.MoveDepartment("backend", "root"))
	views, err = svc.QueryDate("2026-02-01")
	mustOK(t, err)
	if got := joinStrings(views[0].Assignment.Path); got != "root/backend" {
		t.Fatalf("path after move = %q, want root/backend", got)
	}

	// Boundary semantics: end day is exclusive.
	_, err = svc.AddAssignment("bob", "p-lead", "2026-03-01", "2026-04-01")
	mustOK(t, err)
	views, _ = svc.QueryDate("2026-04-01")
	if views[1].Assignment != nil {
		t.Fatal("assignment should have ended on 2026-04-01 (exclusive end)")
	}
	views, _ = svc.QueryDate("2026-03-01")
	if views[1].Assignment == nil {
		t.Fatal("assignment should be active on inclusive start 2026-03-01")
	}
}

func TestDeptCountAfterMove(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.AddAssignment("alice", "p-dev", "2026-01-01", "")
	mustOK(t, err)
	_, err = svc.AddAssignment("bob", "p-lead", "2026-01-01", "")
	mustOK(t, err)

	direct, err := svc.DeptCount("eng", "2026-01-15", false)
	mustOK(t, err)
	if direct != 1 {
		t.Fatalf("direct eng count = %d, want 1", direct)
	}
	recursive, err := svc.DeptCount("eng", "2026-01-15", true)
	mustOK(t, err)
	if recursive != 2 {
		t.Fatalf("recursive eng count = %d, want 2", recursive)
	}

	// Move backend (with alice's post) directly under root.
	mustOK(t, svc.MoveDepartment("backend", "root"))
	recursive, err = svc.DeptCount("eng", "2026-01-15", true)
	mustOK(t, err)
	if recursive != 1 {
		t.Fatalf("recursive eng count after move = %d, want 1", recursive)
	}
	rootCount, err := svc.DeptCount("root", "2026-01-15", true)
	mustOK(t, err)
	if rootCount != 2 {
		t.Fatalf("recursive root count = %d, want 2 (each employee once)", rootCount)
	}
}

func TestPersistenceAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewState(), dir)
	mustOK(t, svc.InitRoot("root", "HQ"))
	mustOK(t, svc.AddDepartment("eng", "Engineering", "root"))
	mustOK(t, svc.AddPost("p1", "Dev", "eng"))
	mustOK(t, svc.AddEmployee("e1", "Alice"))
	_, err := svc.AddAssignment("e1", "p1", "2026-01-01", "2026-02-01")
	mustOK(t, err)

	reloaded, err := LoadState(dir)
	mustOK(t, err)
	svc2 := NewService(reloaded, dir)
	if svc2.State().RootID != "root" {
		t.Fatal("root id not persisted")
	}
	wantErr(t, svc2.InitRoot("other", "Other"), "already initialized")
	hist, err := svc2.History("e1")
	mustOK(t, err)
	if len(hist) != 1 || hist[0].PostID != "p1" {
		t.Fatalf("persisted history wrong: %+v", hist)
	}
}

func joinStrings(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += "/"
		}
		out += x
	}
	return out
}
