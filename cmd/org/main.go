package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"example.com/organization-post-register/internal/org"
)

type request struct {
	Cmd           string `json:"cmd"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	ParentID      string `json:"parent_id"`
	NewParentID   string `json:"new_parent_id"`
	DeptID        string `json:"dept_id"`
	PostID        string `json:"post_id"`
	EmployeeID    string `json:"employee_id"`
	NewPostID     string `json:"new_post_id"`
	NewAssignment string `json:"new_assignment_id"`
	Start         string `json:"start"`
	End           string `json:"end"`
	Date          string `json:"date"`
	Recursive     bool   `json:"recursive"`
}

type response struct {
	OK     bool        `json:"ok"`
	Error  string      `json:"error,omitempty"`
	Result interface{} `json:"result,omitempty"`
}

func main() {
	stateDir := flag.String("state", "", "状态目录")
	flag.Parse()
	if *stateDir == "" {
		fmt.Fprintln(os.Stderr, "必须通过 --state 指定状态目录")
		os.Exit(2)
	}
	svc, err := org.NewService(*stateDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	enc := json.NewEncoder(out)

	for in.Scan() {
		line := in.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			enc.Encode(response{OK: false, Error: "输入不是合法JSON: " + err.Error()})
			continue
		}
		resp := func() (rsp response) {
			defer func() {
				if rec := recover(); rec != nil {
					if ce, ok := rec.(*cmdError); ok {
						rsp = response{OK: false, Error: ce.Error()}
					} else {
						panic(rec)
					}
				}
			}()
			return response{OK: true, Result: dispatch(svc, &req)}
		}()
		enc.Encode(resp)
	}
	if err := in.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dispatch(svc *org.Service, r *request) interface{} {
	switch r.Cmd {
	case "init":
		must(svc.Init(r.ID, r.Name))
	case "dept_add":
		must(svc.AddDepartment(r.ID, r.Name, r.ParentID))
	case "dept_move":
		must(svc.MoveDepartment(r.ID, r.NewParentID))
	case "post_add":
		must(svc.AddPost(r.ID, r.Name, r.DeptID))
	case "employee_add":
		must(svc.AddEmployee(r.ID, r.Name))
	case "assignment_add":
		must(svc.AddAssignment(r.ID, r.EmployeeID, r.PostID, r.Start, r.End))
	case "assignment_end":
		must(svc.EndAssignment(r.ID, r.Date))
	case "transfer":
		next, err := svc.Transfer(r.ID, r.NewAssignment, r.NewPostID, r.Date)
		must(err)
		return next
	case "query_at":
		rows, err := svc.QueryAt(r.Date)
		must(err)
		return rows
	case "dept_count":
		n, err := svc.DeptCount(r.DeptID, r.Date, r.Recursive)
		must(err)
		return map[string]int{"count": n}
	case "history":
		rows, err := svc.History(r.EmployeeID)
		must(err)
		return rows
	default:
		panic(&cmdError{fmt.Sprintf("未知命令 %q", r.Cmd)})
	}
	return map[string]string{"status": "ok"}
}

type cmdError struct{ msg string }

func (e *cmdError) Error() string { return e.msg }

func must(err error) {
	if err != nil {
		panic(&cmdError{err.Error()})
	}
}
