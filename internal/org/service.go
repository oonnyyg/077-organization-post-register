package org

import (
	"fmt"
	"sort"
	"time"
)

const dateLayout = "2006-01-02"

// Service 对状态执行各类命令，变更成功后持久化。
type Service struct {
	dir   string
	state *State
}

func NewService(dir string) (*Service, error) {
	st, err := Load(dir)
	if err != nil {
		return nil, err
	}
	return &Service{dir: dir, state: st}, nil
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("日期不能为空")
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("日期必须为 YYYY-MM-DD 真实日期: %q", s)
	}
	return t, nil
}

func requireID(id, label string) error {
	if id == "" {
		return fmt.Errorf("%s编号不能为空", label)
	}
	return nil
}

func (svc *Service) commit() error {
	return svc.state.save(svc.dir)
}

// Init 初始化根部门，根部门只能指定一次且不可迁移。
func (svc *Service) Init(id, name string) error {
	if err := requireID(id, "部门"); err != nil {
		return err
	}
	if svc.state.Initialized {
		return fmt.Errorf("已初始化，根部门为 %s", svc.state.RootDeptID)
	}
	if _, ok := svc.state.Departments[id]; ok {
		return fmt.Errorf("部门编号 %q 已存在", id)
	}
	svc.state.Departments[id] = &Department{ID: id, Name: name}
	svc.state.RootDeptID = id
	svc.state.Initialized = true
	return svc.commit()
}

// AddDepartment 新增部门，必须引用已有上级。
func (svc *Service) AddDepartment(id, name, parentID string) error {
	if err := requireID(id, "部门"); err != nil {
		return err
	}
	if !svc.state.Initialized {
		return fmt.Errorf("尚未初始化根部门")
	}
	if _, ok := svc.state.Departments[id]; ok {
		return fmt.Errorf("部门编号 %q 已存在", id)
	}
	if _, ok := svc.state.Departments[parentID]; !ok {
		return fmt.Errorf("上级部门 %q 不存在", parentID)
	}
	svc.state.Departments[id] = &Department{ID: id, Name: name, ParentID: parentID}
	return svc.commit()
}

// MoveDepartment 调整部门上级；根部门不可迁移，新上级不能是自身或后代。
func (svc *Service) MoveDepartment(id, newParentID string) error {
	if !svc.state.Initialized {
		return fmt.Errorf("尚未初始化根部门")
	}
	dept, ok := svc.state.Departments[id]
	if !ok {
		return fmt.Errorf("部门 %q 不存在", id)
	}
	if id == svc.state.RootDeptID {
		return fmt.Errorf("根部门不能迁移")
	}
	if _, ok := svc.state.Departments[newParentID]; !ok {
		return fmt.Errorf("上级部门 %q 不存在", newParentID)
	}
	if newParentID == id {
		return fmt.Errorf("部门上级不能是自身")
	}
	if svc.isDescendant(id, newParentID) {
		return fmt.Errorf("部门 %q 是 %q 的后代，不能形成环", newParentID, id)
	}
	dept.ParentID = newParentID
	return svc.commit()
}

// isDescendant 报告 candidate 是否为 ancestorID 的后代（沿父链向上）。
func (svc *Service) isDescendant(ancestorID, candidate string) bool {
	cur := candidate
	for cur != "" {
		d, ok := svc.state.Departments[cur]
		if !ok {
			return false
		}
		if d.ParentID == ancestorID {
			return true
		}
		cur = d.ParentID
	}
	return false
}

// AddPost 登记岗位，岗位固定归属一个已有部门。
func (svc *Service) AddPost(id, name, deptID string) error {
	if err := requireID(id, "岗位"); err != nil {
		return err
	}
	if _, ok := svc.state.Posts[id]; ok {
		return fmt.Errorf("岗位编号 %q 已存在", id)
	}
	if _, ok := svc.state.Departments[deptID]; !ok {
		return fmt.Errorf("部门 %q 不存在", deptID)
	}
	svc.state.Posts[id] = &Post{ID: id, Name: name, DeptID: deptID}
	return svc.commit()
}

// AddEmployee 登记员工。
func (svc *Service) AddEmployee(id, name string) error {
	if err := requireID(id, "员工"); err != nil {
		return err
	}
	if _, ok := svc.state.Employees[id]; ok {
		return fmt.Errorf("员工编号 %q 已存在", id)
	}
	svc.state.Employees[id] = &Employee{ID: id, Name: name}
	return svc.commit()
}

// sortedEmployeeIDs 按编号排序，保证输出稳定。
func (svc *Service) sortedEmployeeIDs() []string {
	ids := make([]string, 0, len(svc.state.Employees))
	for id := range svc.state.Employees {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
