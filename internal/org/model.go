package org

// Assignment 表示一条主岗任职记录，时间区间为 [Start, End)。
// End 为空字符串表示无限期。
type Assignment struct {
	ID         string `json:"id"`
	EmployeeID string `json:"employee_id"`
	PostID     string `json:"post_id"`
	Start      string `json:"start"`
	End        string `json:"end,omitempty"`
}

type Department struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id,omitempty"`
}

type Post struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	DeptID string `json:"dept_id"`
}

type Employee struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// State 是持久化到状态目录的全部数据。
type State struct {
	Initialized bool                   `json:"initialized"`
	RootDeptID  string                 `json:"root_dept_id,omitempty"`
	Departments map[string]*Department `json:"departments"`
	Posts       map[string]*Post       `json:"posts"`
	Employees   map[string]*Employee   `json:"employees"`
	Assignments []*Assignment          `json:"assignments"`
}

func newState() *State {
	return &State{
		Departments: map[string]*Department{},
		Posts:       map[string]*Post{},
		Employees:   map[string]*Employee{},
	}
}
