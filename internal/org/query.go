package org

import (
	"fmt"
	"sort"
	"time"
)

// AssignmentView 是时点查询中单条任职的对外视图。
type AssignmentView struct {
	AssignmentID string   `json:"assignment_id"`
	PostID       string   `json:"post_id"`
	PostName     string   `json:"post_name"`
	DeptID       string   `json:"dept_id"`
	DeptName     string   `json:"dept_name"`
	Path         []string `json:"path"`
	PathNames    []string `json:"path_names"`
}

// EmployeeAt 描述某员工在指定日期的主岗归属。
type EmployeeAt struct {
	EmployeeID string          `json:"employee_id"`
	Name       string          `json:"name"`
	Current    *AssignmentView `json:"current"`
}

// departmentPath 返回当前部门树中从根到 deptID 的编号路径。
// 若链断裂（理论上不应发生），返回错误。
func (svc *Service) departmentPath(deptID string) ([]string, []string, error) {
	var ids, names []string
	cur := deptID
	for cur != "" {
		d, ok := svc.state.Departments[cur]
		if !ok {
			return nil, nil, fmt.Errorf("部门 %q 在当前部门树中缺失", cur)
		}
		ids = append(ids, d.ID)
		names = append(names, d.Name)
		if cur == svc.state.RootDeptID {
			break
		}
		cur = d.ParentID
	}
	for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
		ids[i], ids[j] = ids[j], ids[i]
		names[i], names[j] = names[j], names[i]
	}
	return ids, names, nil
}

func (svc *Service) viewAt(a *Assignment) (*AssignmentView, error) {
	post := svc.state.Posts[a.PostID]
	dept := svc.state.Departments[post.DeptID]
	ids, names, err := svc.departmentPath(dept.ID)
	if err != nil {
		return nil, err
	}
	return &AssignmentView{
		AssignmentID: a.ID,
		PostID:       post.ID,
		PostName:     post.Name,
		DeptID:       dept.ID,
		DeptName:     dept.Name,
		Path:         ids,
		PathNames:    names,
	}, nil
}

// activeAt 返回员工在 date 当日有效的主岗任职；无则 nil。
func (svc *Service) activeAt(employeeID string, d time.Time) *Assignment {
	var found *Assignment
	for _, a := range svc.state.Assignments {
		if a.EmployeeID != employeeID {
			continue
		}
		st, _ := parseDate(a.Start)
		if d.Before(st) {
			continue
		}
		if a.End != "" {
			et, _ := parseDate(a.End)
			if !d.Before(et) {
				continue
			}
		}
		found = a
	}
	return found
}

// QueryAt 查询某日全部员工的主岗与部门路径；无任职者 current 为 null。
func (svc *Service) QueryAt(date string) ([]EmployeeAt, error) {
	d, err := parseDate(date)
	if err != nil {
		return nil, err
	}
	out := make([]EmployeeAt, 0, len(svc.state.Employees))
	for _, id := range svc.sortedEmployeeIDs() {
		emp := svc.state.Employees[id]
		row := EmployeeAt{EmployeeID: id, Name: emp.Name}
		if a := svc.activeAt(id, d); a != nil {
			v, err := svc.viewAt(a)
			if err != nil {
				return nil, err
			}
			row.Current = v
		}
		out = append(out, row)
	}
	return out, nil
}

func (svc *Service) descendantDepts(deptID string, recursive bool) map[string]bool {
	result := map[string]bool{deptID: true}
	if !recursive {
		return result
	}
	children := map[string][]string{}
	for id, d := range svc.state.Departments {
		if d.ParentID != "" {
			children[d.ParentID] = append(children[d.ParentID], id)
		}
	}
	queue := []string{deptID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range children[cur] {
			if !result[c] {
				result[c] = true
				queue = append(queue, c)
			}
		}
	}
	return result
}

// DeptCount 统计某日部门人数；recursive 为 true 时包含全部后代部门。
// 每名员工当日最多一条主岗，因此天然只计一次。
func (svc *Service) DeptCount(deptID, date string, recursive bool) (int, error) {
	if _, ok := svc.state.Departments[deptID]; !ok {
		return 0, fmt.Errorf("部门 %q 不存在", deptID)
	}
	d, err := parseDate(date)
	if err != nil {
		return 0, err
	}
	depts := svc.descendantDepts(deptID, recursive)
	count := 0
	for _, id := range svc.sortedEmployeeIDs() {
		a := svc.activeAt(id, d)
		if a == nil {
			continue
		}
		if depts[svc.state.Posts[a.PostID].DeptID] {
			count++
		}
	}
	return count, nil
}

// History 返回某员工按开始日排序的全部任职记录。
func (svc *Service) History(employeeID string) ([]*Assignment, error) {
	if _, ok := svc.state.Employees[employeeID]; !ok {
		return nil, fmt.Errorf("员工 %q 不存在", employeeID)
	}
	var rows []*Assignment
	for _, a := range svc.state.Assignments {
		if a.EmployeeID == employeeID {
			cp := *a
			rows = append(rows, &cp)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Start == rows[j].Start {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].Start < rows[j].Start
	})
	return rows, nil
}
