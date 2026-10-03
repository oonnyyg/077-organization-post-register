package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, stateDir, input string) []map[string]any {
	t.Helper()
	cmd := exec.Command("go", "run", "./cmd/org", "--state", stateDir)
	cmd.Dir = filepath.Join("..", "..")
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli failed: %v\n%s", err, out)
	}
	var res []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("invalid response line %q: %v", line, err)
		}
		res = append(res, m)
	}
	return res
}

func TestCLIEndToEnd(t *testing.T) {
	stateDir := t.TempDir()
	input := strings.Join([]string{
		`{"cmd":"init_root","id":"root","name":"HQ"}`,
		`{"cmd":"dept_add","id":"eng","name":"Eng","parent_id":"root"}`,
		`{"cmd":"post_add","id":"p1","name":"Dev","dept_id":"eng"}`,
		`{"cmd":"employee_add","id":"e1","name":"Alice"}`,
		`{"cmd":"assignment_add","employee_id":"e1","post_id":"p1","start":"2026-01-01","end":"2026-05-01"}`,
		`{"cmd":"transfer","id":"a1","new_post_id":"p1","date":"2026-03-01"}`,
		`{"cmd":"transfer","id":"a1","new_post_id":"p1","date":"2026-03-01"}`,
		`{"cmd":"query_date","date":"2026-04-01"}`,
		``,
	}, "\n")

	res := runCLI(t, stateDir, input)
	if len(res) != 8 {
		t.Fatalf("got %d response lines, want 8 (blank line ignored)", len(res))
	}
	for i := 0; i < 5; i++ {
		if res[i]["ok"] != true {
			t.Fatalf("line %d expected ok: %v", i, res[i])
		}
	}
	// First transfer succeeds shortening a1 to end on 2026-03-01; a second
	// transfer on that same date then fails (must be strictly before end).
	if res[5]["ok"] != true {
		t.Fatalf("line 5 expected successful transfer: %v", res[5])
	}
	if res[6]["ok"] != false || !strings.Contains(asString(res[6]["error"]), "strictly before") {
		t.Fatalf("line 6 expected boundary error: %v", res[6])
	}
	if res[7]["ok"] != true {
		t.Fatalf("query failed: %v", res[7])
	}

	// Restart: history is persisted.
	res2 := runCLI(t, stateDir, `{"cmd":"history","employee_id":"e1"}`+"\n")
	if res2[0]["ok"] != true {
		t.Fatalf("history after restart failed: %v", res2[0])
	}
	list, _ := res2[0]["assignments"].([]any)
	if len(list) != 2 {
		t.Fatalf("after restart assignments = %d, want 2", len(list))
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
