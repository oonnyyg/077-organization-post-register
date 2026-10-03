package org

import (
	"fmt"
	"time"
)

var farFuture = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

func intervalEnd(a *Assignment) time.Time {
	if a.End == "" {
		return farFuture
	}
	t, _ := parseDate(a.End)
	return t
}

func intervalStart(a *Assignment) time.Time {
	t, _ := parseDate(a.Start)
	return t
}

// overlaps 判断两个半开区间 [start,end) 是否重叠；首尾相接不算重叠。
func overlaps(a, b *Assignment) bool {
	return intervalStart(a).Before(intervalEnd(b)) &&
		intervalStart(b).Before(intervalEnd(a))
}

// AddAssignment 录入一条任职。同一员工的主岗任职不能时间重叠。
func (svc *Service) AddAssignment(id, employeeID, postID, start, end string) error {
	if err := requireID(id, "任职"); err != nil {
		return err
	}
	for _, a := range svc.state.Assignments {
		if a.ID == id {
			return fmt.Errorf("任职编号 %q 已存在", id)
		}
	}
	emp, ok := svc.state.Employees[employeeID]
	if !ok {
		return fmt.Errorf("员工 %q 不存在", employeeID)
	}
	_ = emp
	post, ok := svc.state.Posts[postID]
	if !ok {
		return fmt.Errorf("岗位 %q 不存在", postID)
	}
	_ = post
	st, err := parseDate(start)
	if err != nil {
		return err
	}
	if end != "" {
		et, err := parseDate(end)
		if err != nil {
			return err
		}
		if !et.After(st) {
			return fmt.Errorf("结束日 %s 必须晚于开始日 %s", end, start)
		}
	}
	candidate := &Assignment{ID: id, EmployeeID: employeeID, PostID: postID, Start: start, End: end}
	for _, a := range svc.state.Assignments {
		if a.EmployeeID != employeeID {
			continue
		}
		if overlaps(candidate, a) {
			return fmt.Errorf("员工 %q 在 %s 已有任职 %q，时间区间重叠", employeeID, overlapRange(candidate, a), a.ID)
		}
	}
	svc.state.Assignments = append(svc.state.Assignments, candidate)
	return svc.commit()
}

func overlapRange(a, b *Assignment) string {
	return fmt.Sprintf("%s~%s 与 %s~%s", a.Start, openEnd(a.End), b.Start, openEnd(b.End))
}

func openEnd(end string) string {
	if end == "" {
		return "至今"
	}
	return end
}

// EndAssignment 在指定日期结束一条仍在进行中的任职。
// 日期须严格晚于开始日；对已有结束日的任职，日期须早于其结束日。
func (svc *Service) EndAssignment(id, date string) error {
	a, err := svc.findAssignment(id)
	if err != nil {
		return err
	}
	d, err := parseDate(date)
	if err != nil {
		return err
	}
	if !d.After(intervalStart(a)) {
		return fmt.Errorf("结束日期 %s 必须严格晚于开始日 %s", date, a.Start)
	}
	if a.End != "" && !d.Before(intervalEnd(a)) {
		return fmt.Errorf("结束日期 %s 必须早于已有结束日 %s", date, a.End)
	}
	a.End = date
	return svc.commit()
}

func (svc *Service) findAssignment(id string) (*Assignment, error) {
	if err := requireID(id, "任职"); err != nil {
		return nil, err
	}
	for _, a := range svc.state.Assignments {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, fmt.Errorf("任职 %q 不存在", id)
}

// Transfer 将任职 id 在 date 当日结束，并同日开始一条新岗位任职。
// 新任职沿用原结束日。全部校验通过后才同时更新，避免提前截断原任职。
func (svc *Service) Transfer(id, newAssignmentID, newPostID, date string) (*Assignment, error) {
	a, err := svc.findAssignment(id)
	if err != nil {
		return nil, err
	}
	if err := requireID(newAssignmentID, "新任职"); err != nil {
		return nil, err
	}
	for _, x := range svc.state.Assignments {
		if x.ID == newAssignmentID {
			return nil, fmt.Errorf("任职编号 %q 已存在", newAssignmentID)
		}
	}
	if _, ok := svc.state.Posts[newPostID]; !ok {
		return nil, fmt.Errorf("岗位 %q 不存在", newPostID)
	}
	d, err := parseDate(date)
	if err != nil {
		return nil, err
	}
	if !d.After(intervalStart(a)) {
		return nil, fmt.Errorf("调岗日期 %s 必须严格晚于开始日 %s", date, a.Start)
	}
	if a.End == "" {
		return nil, fmt.Errorf("任职 %q 无结束日，无法按“沿用原结束日”调岗", id)
	}
	if !d.Before(intervalEnd(a)) {
		return nil, fmt.Errorf("调岗日期 %s 必须早于已有结束日 %s", date, a.End)
	}

	next := &Assignment{
		ID:         newAssignmentID,
		EmployeeID: a.EmployeeID,
		PostID:     newPostID,
		Start:      date,
		End:        a.End,
	}
	for _, x := range svc.state.Assignments {
		if x == a || x.EmployeeID != a.EmployeeID {
			continue
		}
		if overlaps(next, x) {
			return nil, fmt.Errorf("员工 %q 的新任职与已有任职 %q 时间重叠（%s~%s 与 %s~%s）",
				a.EmployeeID, x.ID, date, openEnd(next.End), x.Start, openEnd(x.End))
		}
	}

	a.End = date
	svc.state.Assignments = append(svc.state.Assignments, next)
	if err := svc.commit(); err != nil {
		// 提交失败时回滚内存状态，保持与磁盘一致。
		a.End = next.End
		svc.state.Assignments = svc.state.Assignments[:len(svc.state.Assignments)-1]
		return nil, err
	}
	return next, nil
}
