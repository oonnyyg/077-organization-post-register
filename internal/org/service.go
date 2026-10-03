package org

import "sort"

// Service holds the in-memory state and the directory it is persisted to.
type Service struct {
	state *State
	dir   string
}

func NewService(state *State, dir string) *Service {
	return &Service{state: state, dir: dir}
}

func (s *Service) State() *State { return s.state }

func requireID(kind, id string) error {
	if id == "" {
		return failf("%s id must be non-empty", kind)
	}
	return nil
}

// InitRoot defines the immutable root department. It may run only once.
func (s *Service) InitRoot(id, name string) error {
	if err := requireID("department", id); err != nil {
		return err
	}
	if s.state.RootID != "" {
		return failf("root department already initialized as %q", s.state.RootID)
	}
	if _, exists := s.state.Departments[id]; exists {
		return failf("department %q already exists", id)
	}
	s.state.RootID = id
	s.state.Departments[id] = &Department{ID: id, Name: name}
	return s.state.Save(s.dir)
}

// AddDepartment registers a non-root department under an existing parent.
func (s *Service) AddDepartment(id, name, parentID string) error {
	if err := requireID("department", id); err != nil {
		return err
	}
	if s.state.RootID == "" {
		return failf("root department must be initialized first")
	}
	if _, exists := s.state.Departments[id]; exists {
		return failf("department %q already exists", id)
	}
	parent, ok := s.state.Departments[parentID]
	if !ok {
		return failf("parent department %q not found", parentID)
	}
	_ = parent
	s.state.Departments[id] = &Department{ID: id, Name: name, ParentID: parentID}
	return s.state.Save(s.dir)
}

// MoveDepartment re-parents a department. The root cannot move, and the
// new parent must exist and must not be the department itself or a descendant.
func (s *Service) MoveDepartment(id, newParentID string) error {
	if s.state.RootID == "" {
		return failf("root department must be initialized first")
	}
	dept, ok := s.state.Departments[id]
	if !ok {
		return failf("department %q not found", id)
	}
	if id == s.state.RootID {
		return failf("root department %q cannot be moved", id)
	}
	if _, ok := s.state.Departments[newParentID]; !ok {
		return failf("parent department %q not found", newParentID)
	}
	if newParentID == id {
		return failf("department %q cannot be its own parent", id)
	}
	if s.isDescendant(id, newParentID) {
		return failf("cannot move department %q under its own descendant %q", id, newParentID)
	}
	dept.ParentID = newParentID
	return s.state.Save(s.dir)
}

// isDescendant reports whether candidate is strictly below ancestor.
func (s *Service) isDescendant(ancestor, candidate string) bool {
	cur := candidate
	for cur != "" {
		if cur == ancestor {
			return true
		}
		d, ok := s.state.Departments[cur]
		if !ok {
			return false
		}
		cur = d.ParentID
	}
	return false
}

// AddPost registers a post permanently attached to an existing department.
func (s *Service) AddPost(id, name, deptID string) error {
	if err := requireID("post", id); err != nil {
		return err
	}
	if _, exists := s.state.Posts[id]; exists {
		return failf("post %q already exists", id)
	}
	if _, ok := s.state.Departments[deptID]; !ok {
		return failf("department %q not found", deptID)
	}
	s.state.Posts[id] = &Post{ID: id, Name: name, DeptID: deptID}
	return s.state.Save(s.dir)
}

// AddEmployee registers an employee who may later hold zero or more posts.
func (s *Service) AddEmployee(id, name string) error {
	if err := requireID("employee", id); err != nil {
		return err
	}
	if _, exists := s.state.Employees[id]; exists {
		return failf("employee %q already exists", id)
	}
	s.state.Employees[id] = &Employee{ID: id, Name: name}
	return s.state.Save(s.dir)
}

// SortedDepartmentIDs returns department IDs sorted for stable output.
func (s *Service) SortedDepartmentIDs() []string {
	ids := make([]string, 0, len(s.state.Departments))
	for id := range s.state.Departments {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
