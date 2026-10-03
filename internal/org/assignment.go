package org

import (
	"sort"
	"strconv"
	"time"
)

func nextAssignmentID(seq int) string {
	return "a" + strconv.Itoa(seq)
}

// AddAssignment records a half-open [start, end) primary-post assignment.
// An empty end means open-ended. Adjacent intervals for the same employee
// are allowed; overlaps are rejected.
func (s *Service) AddAssignment(employeeID, postID, start, end string) (string, error) {
	if _, ok := s.state.Employees[employeeID]; !ok {
		return "", failf("employee %q not found", employeeID)
	}
	if _, ok := s.state.Posts[postID]; !ok {
		return "", failf("post %q not found", postID)
	}
	startT, err := ParseDate(start)
	if err != nil {
		return "", err
	}
	var endT time.Time
	if end != "" {
		endT, err = ParseDate(end)
		if err != nil {
			return "", err
		}
		if !endT.After(startT) {
			return "", failf("end date %s must be later than start date %s", end, start)
		}
	}
	if err := s.checkOverlap(employeeID, startT, endT, ""); err != nil {
		return "", err
	}
	s.state.AssignmentSeq++
	id := nextAssignmentID(s.state.AssignmentSeq)
	s.state.Assignments[id] = &Assignment{
		ID:         id,
		EmployeeID: employeeID,
		PostID:     postID,
		Start:      start,
		End:        end,
	}
	if err := s.state.Save(s.dir); err != nil {
		return "", err
	}
	return id, nil
}

// Transfer ends assignmentID on the given date and starts a new assignment
// on newPostID the same day. The date must be strictly after the assignment
// start and strictly before its current end. The new assignment inherits the
// original end. Both records are updated only after every check passes.
func (s *Service) Transfer(assignmentID, newPostID, date string) (string, error) {
	a, ok := s.state.Assignments[assignmentID]
	if !ok {
		return "", failf("assignment %q not found", assignmentID)
	}
	if _, ok := s.state.Posts[newPostID]; !ok {
		return "", failf("post %q not found", newPostID)
	}
	startT, err := ParseDate(a.Start)
	if err != nil {
		return "", err
	}
	if a.End == "" {
		return "", failf("assignment %q is open-ended and cannot be transferred", assignmentID)
	}
	endT, err := ParseDate(a.End)
	if err != nil {
		return "", err
	}
	dateT, err := ParseDate(date)
	if err != nil {
		return "", err
	}
	if !dateT.After(startT) {
		return "", failf("transfer date %s must be strictly after assignment start %s", date, a.Start)
	}
	if !dateT.Before(endT) {
		return "", failf("transfer date %s must be strictly before assignment end %s", date, a.End)
	}
	// Validate the new interval against every other assignment of the employee.
	if err := s.checkOverlap(a.EmployeeID, dateT, endT, a.ID); err != nil {
		return "", err
	}

	// All checks passed: capture the original end, mutate both records, persist.
	originalEnd := a.End
	s.state.AssignmentSeq++
	newID := nextAssignmentID(s.state.AssignmentSeq)
	a.End = date
	s.state.Assignments[newID] = &Assignment{
		ID:         newID,
		EmployeeID: a.EmployeeID,
		PostID:     newPostID,
		Start:      date,
		End:        originalEnd,
	}
	if err := s.state.Save(s.dir); err != nil {
		return "", err
	}
	return newID, nil
}

// History returns all assignments of an employee ordered by start date.
func (s *Service) History(employeeID string) ([]*Assignment, error) {
	if _, ok := s.state.Employees[employeeID]; !ok {
		return nil, failf("employee %q not found", employeeID)
	}
	var list []*Assignment
	for _, a := range s.state.Assignments {
		if a.EmployeeID == employeeID {
			list = append(list, a)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Start != list[j].Start {
			return list[i].Start < list[j].Start
		}
		return list[i].ID < list[j].ID
	})
	return list, nil
}

// checkOverlap reports whether [start, end) clashes with any assignment of
// employeeID other than excludeID. A zero end means open-ended.
func (s *Service) checkOverlap(employeeID string, start, end time.Time, excludeID string) error {
	for _, a := range s.state.Assignments {
		if a.EmployeeID != employeeID || a.ID == excludeID {
			continue
		}
		aStart, err := ParseDate(a.Start)
		if err != nil {
			return err
		}
		var aEnd time.Time
		if a.End != "" {
			aEnd, err = ParseDate(a.End)
			if err != nil {
				return err
			}
		}
		if intervalsOverlap(start, end, aStart, aEnd) {
			return failf("overlaps with existing assignment %q [%s, %s)", a.ID, a.Start, a.End)
		}
	}
	return nil
}
