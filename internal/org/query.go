package org

import (
	"sort"
	"time"
)

// AssignmentView is one employee's primary post on a queried date.
type AssignmentView struct {
	AssignmentID string   `json:"assignment_id"`
	PostID       string   `json:"post_id"`
	PostName     string   `json:"post_name"`
	DeptID       string   `json:"dept_id"`
	DeptName     string   `json:"dept_name"`
	Path         []string `json:"path"`
	PathNames    []string `json:"path_names"`
}

// EmployeeView pairs an employee with their (possibly empty) assignment.
type EmployeeView struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Assignment *AssignmentView `json:"assignment,omitempty"`
}

// QueryDate resolves every employee's primary post on date. Employees with no
// active assignment are kept with a nil assignment. Department paths are
// always resolved against the current department tree.
func (s *Service) QueryDate(date string) ([]*EmployeeView, error) {
	if s.state.RootID == "" {
		return nil, failf("root department must be initialized first")
	}
	dateT, err := ParseDate(date)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(s.state.Employees))
	for id := range s.state.Employees {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	views := make([]*EmployeeView, 0, len(ids))
	for _, id := range ids {
		emp := s.state.Employees[id]
		view := &EmployeeView{ID: emp.ID, Name: emp.Name}
		if active := s.activeAssignment(emp.ID, dateT); active != nil {
			av, err := s.buildAssignmentView(active)
			if err != nil {
				return nil, err
			}
			view.Assignment = av
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) activeAssignment(employeeID string, date time.Time) *Assignment {
	var match *Assignment
	for _, a := range s.state.Assignments {
		if a.EmployeeID != employeeID {
			continue
		}
		start, err := ParseDate(a.Start)
		if err != nil {
			continue
		}
		if date.Before(start) {
			continue
		}
		if a.End != "" {
			end, err := ParseDate(a.End)
			if err != nil {
				continue
			}
			if !date.Before(end) {
				continue
			}
		}
		match = a
	}
	return match
}

func (s *Service) buildAssignmentView(a *Assignment) (*AssignmentView, error) {
	post := s.state.Posts[a.PostID]
	dept := s.state.Departments[post.DeptID]
	path, pathNames, err := s.departmentPath(dept.ID)
	if err != nil {
		return nil, err
	}
	return &AssignmentView{
		AssignmentID: a.ID,
		PostID:       post.ID,
		PostName:     post.Name,
		DeptID:       dept.ID,
		DeptName:     dept.Name,
		Path:         path,
		PathNames:    pathNames,
	}, nil
}

// departmentPath returns IDs and names from the root down to deptID using
// the current tree only.
func (s *Service) departmentPath(deptID string) ([]string, []string, error) {
	var revIDs, revNames []string
	cur := deptID
	for cur != "" {
		d, ok := s.state.Departments[cur]
		if !ok {
			return nil, nil, failf("department %q missing from tree", cur)
		}
		revIDs = append(revIDs, d.ID)
		revNames = append(revNames, d.Name)
		if cur == s.state.RootID {
			break
		}
		cur = d.ParentID
	}
	reverse(revIDs)
	reverse(revNames)
	return revIDs, revNames, nil
}

func reverse(xs []string) {
	for i, j := 0, len(xs)-1; i < j; i, j = i+1, j-1 {
		xs[i], xs[j] = xs[j], xs[i]
	}
}

// DeptCount counts distinct employees holding a primary post in the
// department on date. recursive=true includes all descendant departments.
func (s *Service) DeptCount(deptID, date string, recursive bool) (int, error) {
	if s.state.RootID == "" {
		return 0, failf("root department must be initialized first")
	}
	if _, ok := s.state.Departments[deptID]; !ok {
		return 0, failf("department %q not found", deptID)
	}
	dateT, err := ParseDate(date)
	if err != nil {
		return 0, err
	}
	counted := map[string]bool{}
	depts := map[string]bool{deptID: true}
	if recursive {
		for id := range s.descendants(deptID) {
			depts[id] = true
		}
	}
	for _, a := range s.state.Assignments {
		if counted[a.EmployeeID] {
			continue
		}
		post, ok := s.state.Posts[a.PostID]
		if !ok || !depts[post.DeptID] {
			continue
		}
		start, perr := ParseDate(a.Start)
		if perr != nil || dateT.Before(start) {
			continue
		}
		if a.End != "" {
			end, perr := ParseDate(a.End)
			if perr != nil || !dateT.Before(end) {
				continue
			}
		}
		counted[a.EmployeeID] = true
	}
	return len(counted), nil
}

// descendants returns the strict descendant closure of deptID.
func (s *Service) descendants(deptID string) map[string]bool {
	children := map[string][]string{}
	for id, d := range s.state.Departments {
		if d.ParentID != "" {
			children[d.ParentID] = append(children[d.ParentID], id)
		}
	}
	out := map[string]bool{}
	frontier := append([]string{}, children[deptID]...)
	for len(frontier) > 0 {
		cur := frontier[len(frontier)-1]
		frontier = frontier[:len(frontier)-1]
		if out[cur] {
			continue
		}
		out[cur] = true
		frontier = append(frontier, children[cur]...)
	}
	return out
}
