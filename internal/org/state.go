package org

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Department is a node in the single-root department tree.
// ParentID is empty for the root department.
type Department struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}

// Post belongs to exactly one department for its whole lifetime.
type Post struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	DeptID string `json:"dept_id"`
}

type Employee struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Assignment is a half-open interval [Start, End) of an employee's
// primary post. An empty End means the assignment is open-ended.
type Assignment struct {
	ID         string `json:"id"`
	EmployeeID string `json:"employee_id"`
	PostID     string `json:"post_id"`
	Start      string `json:"start"`
	End        string `json:"end"`
}

// State is the complete persisted data of the organization.
type State struct {
	RootID        string                 `json:"root_id"`
	Departments   map[string]*Department `json:"departments"`
	Posts         map[string]*Post       `json:"posts"`
	Employees     map[string]*Employee   `json:"employees"`
	Assignments   map[string]*Assignment `json:"assignments"`
	AssignmentSeq int                    `json:"assignment_seq"`
}

const stateFileName = "state.json"

// NewState returns an empty, initialized state.
func NewState() *State {
	return &State{
		Departments: map[string]*Department{},
		Posts:       map[string]*Post{},
		Employees:   map[string]*Employee{},
		Assignments: map[string]*Assignment{},
	}
}

// LoadState reads the state from stateDir. A missing file means a fresh state.
func LoadState(stateDir string) (*State, error) {
	path := filepath.Join(stateDir, stateFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewState(), nil
	}
	if err != nil {
		return nil, err
	}
	st := NewState()
	if err := json.Unmarshal(data, st); err != nil {
		return nil, fmt.Errorf("corrupt state file: %w", err)
	}
	if st.Departments == nil {
		st.Departments = map[string]*Department{}
	}
	if st.Posts == nil {
		st.Posts = map[string]*Post{}
	}
	if st.Employees == nil {
		st.Employees = map[string]*Employee{}
	}
	if st.Assignments == nil {
		st.Assignments = map[string]*Assignment{}
	}
	return st, nil
}

// Save writes the state atomically (temporary file in the same directory,
// then rename) so a crash cannot leave a truncated state file.
func (s *State) Save(stateDir string) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(stateDir, stateFileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
