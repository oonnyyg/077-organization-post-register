package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"example.com/organization-post-register/internal/org"
)

// Request is one JSON line on stdin.
type Request struct {
	Cmd        string `json:"cmd"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	ParentID   string `json:"parent_id"`
	NewParent  string `json:"new_parent_id"`
	DeptID     string `json:"dept_id"`
	EmployeeID string `json:"employee_id"`
	PostID     string `json:"post_id"`
	NewPostID  string `json:"new_post_id"`
	Start      string `json:"start"`
	End        string `json:"end"`
	Date       string `json:"date"`
	Recursive  bool   `json:"recursive"`
}

func main() {
	stateDir := flag.String("state", "", "directory holding state.json")
	flag.Parse()
	if *stateDir == "" {
		fmt.Fprintln(os.Stderr, "missing required --state directory")
		os.Exit(2)
	}

	state, err := org.LoadState(*stateDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load state:", err)
		os.Exit(1)
	}
	svc := org.NewService(state, *stateDir)

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for in.Scan() {
		line := in.Bytes()
		if len(bytesTrimSpace(line)) == 0 {
			continue
		}
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			writeError(out, "", "invalid JSON: "+err.Error())
			continue
		}
		resp := dispatch(svc, &req)
		writeJSON(out, resp)
	}
	if err := in.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "read stdin:", err)
		os.Exit(1)
	}
}

func dispatch(svc *org.Service, req *Request) map[string]any {
	switch req.Cmd {
	case "init_root":
		if err := svc.InitRoot(req.ID, req.Name); err != nil {
			return errResponse(err)
		}
		return map[string]any{"ok": true, "cmd": req.Cmd, "id": req.ID}
	case "dept_add":
		if err := svc.AddDepartment(req.ID, req.Name, req.ParentID); err != nil {
			return errResponse(err)
		}
		return map[string]any{"ok": true, "cmd": req.Cmd, "id": req.ID}
	case "dept_move":
		if err := svc.MoveDepartment(req.ID, req.NewParent); err != nil {
			return errResponse(err)
		}
		return map[string]any{"ok": true, "cmd": req.Cmd, "id": req.ID, "parent_id": req.NewParent}
	case "post_add":
		if err := svc.AddPost(req.ID, req.Name, req.DeptID); err != nil {
			return errResponse(err)
		}
		return map[string]any{"ok": true, "cmd": req.Cmd, "id": req.ID}
	case "employee_add":
		if err := svc.AddEmployee(req.ID, req.Name); err != nil {
			return errResponse(err)
		}
		return map[string]any{"ok": true, "cmd": req.Cmd, "id": req.ID}
	case "assignment_add":
		id, err := svc.AddAssignment(req.EmployeeID, req.PostID, req.Start, req.End)
		if err != nil {
			return errResponse(err)
		}
		return map[string]any{"ok": true, "cmd": req.Cmd, "id": id}
	case "transfer":
		id, err := svc.Transfer(req.ID, req.NewPostID, req.Date)
		if err != nil {
			return errResponse(err)
		}
		return map[string]any{"ok": true, "cmd": req.Cmd, "id": id}
	case "history":
		list, err := svc.History(req.EmployeeID)
		if err != nil {
			return errResponse(err)
		}
		return map[string]any{"ok": true, "cmd": req.Cmd, "employee_id": req.EmployeeID, "assignments": list}
	case "query_date":
		views, err := svc.QueryDate(req.Date)
		if err != nil {
			return errResponse(err)
		}
		return map[string]any{"ok": true, "cmd": req.Cmd, "date": req.Date, "employees": views}
	case "dept_count":
		n, err := svc.DeptCount(req.DeptID, req.Date, req.Recursive)
		if err != nil {
			return errResponse(err)
		}
		return map[string]any{"ok": true, "cmd": req.Cmd, "dept_id": req.DeptID, "date": req.Date, "recursive": req.Recursive, "count": n}
	default:
		return map[string]any{"ok": false, "error": "unknown cmd: " + req.Cmd}
	}
}

func errResponse(err error) map[string]any {
	var ve *org.ValidationError
	if errors.As(err, &ve) {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": false, "error": "internal error: " + err.Error()}
}

func writeError(out *bufio.Writer, cmd, msg string) {
	writeJSON(out, map[string]any{"ok": false, "cmd": cmd, "error": msg})
}

func writeJSON(out *bufio.Writer, v map[string]any) {
	data, err := json.Marshal(v)
	if err != nil {
		data = []byte(`{"ok":false,"error":"failed to encode response"}`)
	}
	out.Write(data)
	out.WriteByte('\n')
	out.Flush()
}

func bytesTrimSpace(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && (b[start] == ' ' || b[start] == '\t' || b[start] == '\r') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t' || b[end-1] == '\r') {
		end--
	}
	return b[start:end]
}
