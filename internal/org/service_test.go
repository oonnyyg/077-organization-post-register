package org

import "testing"

func setup(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mustInit(t, svc)
	return svc
}

func mustInit(t *testing.T, svc *Service) {
	t.Helper()
	if err := svc.Init("ROOT", "根"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddDepartment("ENG", "工程部", "ROOT"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddDepartment("BE", "后端", "ENG"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddDepartment("FE", "前端", "ENG"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddPost("P1", "工程师", "BE"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddPost("P2", "架构师", "ENG"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddPost("P3", "前端工程师", "FE"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddEmployee("E1", "张三"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddEmployee("E2", "李四"); err != nil {
		t.Fatal(err)
	}
}

func TestAdjacentAssignmentsAllowed(t *testing.T) {
	svc := setup(t)
	if err := svc.AddAssignment("A1", "E1", "P1", "2026-01-01", "2026-03-01"); err != nil {
		t.Fatal(err)
	}
	// 首尾相接：上一段结束日即下一段开始日，允许。
	if err := svc.AddAssignment("A2", "E1", "P2", "2026-03-01", ""); err != nil {
		t.Fatalf("相邻任职应被允许: %v", err)
	}
	// 重叠一天即拒绝。
	if err := svc.AddAssignment("A3", "E2", "P3", "2026-02-01", "2026-02-01"); err == nil {
		t.Fatal("结束日等于开始日应报错")
	}
}

func TestOverlapRejected(t *testing.T) {
	svc := setup(t)
	if err := svc.AddAssignment("A1", "E1", "P1", "2026-01-01", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddAssignment("A2", "E1", "P2", "2026-02-01", "2026-03-01"); err == nil {
		t.Fatal("与无限期任职重叠应报错")
	}
}

func TestTransferFailureKeepsOriginal(t *testing.T) {
	svc := setup(t)
	if err := svc.AddAssignment("A1", "E1", "P1", "2026-01-01", "2026-12-01"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transfer("A1", "A2", "P2", "2026-06-01"); err != nil {
		t.Fatal(err)
	}
	a, _ := svc.findAssignment("A1")
	if a.End != "2026-06-01" {
		t.Fatalf("调岗失败不得提前截断原任职, end=%q", a.End)
	}
	n := len(svc.state.Assignments)
	for _, x := range svc.state.Assignments {
		if x.ID == "A3" {
			t.Fatal("失败的调岗不得产生新任职记录")
		}
	}
	// 未知岗位同样失败且不改状态。
	if _, err := svc.Transfer("A1", "A3", "NOPE", "2026-03-01"); err == nil {
		t.Fatal("未知岗位应报错")
	}
	// 日期不严格晚于开始日，失败且不截断。
	if _, err := svc.Transfer("A1", "A3", "P3", "2026-01-01"); err == nil {
		t.Fatal("调岗日期等于开始日应报错")
	}
	// 日期不早于已有结束日，失败且不截断。
	if _, err := svc.Transfer("A1", "A3", "P3", "2026-06-01"); err == nil {
		t.Fatal("调岗日期等于结束日应报错")
	}
	// 重复的新任职编号，失败且不截断。
	if _, err := svc.Transfer("A1", "A2", "P3", "2026-03-01"); err == nil {
		t.Fatal("新任职编号重复应报错")
	}
	if a, _ := svc.findAssignment("A1"); a.End != "2026-06-01" {
		t.Fatalf("边界日期失败后原结束日被改动: %q", a.End)
	}
	if len(svc.state.Assignments) != n {
		t.Fatal("失败后任职记录数量发生变化")
	}
}

func TestTransferSuccess(t *testing.T) {
	svc := setup(t)
	if err := svc.AddAssignment("A1", "E1", "P1", "2026-01-01", "2026-06-01"); err != nil {
		t.Fatal(err)
	}
	next, err := svc.Transfer("A1", "A2", "P2", "2026-03-01")
	if err != nil {
		t.Fatal(err)
	}
	if next.End != "2026-06-01" || next.Start != "2026-03-01" || next.PostID != "P2" {
		t.Fatalf("新任职未沿用原结束日: %+v", next)
	}
	rows, _ := svc.QueryAt("2026-02-28")
	if rows[0].Current.PostID != "P1" {
		t.Fatal("调岗日前应仍在原岗位")
	}
	rows, _ = svc.QueryAt("2026-03-01")
	if rows[0].Current.PostID != "P2" {
		t.Fatal("调岗日当日应在新岗位")
	}
}

func TestMoveDepartmentRules(t *testing.T) {
	svc := setup(t)
	if err := svc.MoveDepartment("ROOT", "ENG"); err == nil {
		t.Fatal("根部门不能迁移")
	}
	if err := svc.MoveDepartment("ENG", "ENG"); err == nil {
		t.Fatal("不能以自身为上级")
	}
	if err := svc.MoveDepartment("ENG", "BE"); err == nil {
		t.Fatal("不能以后代部门为上级")
	}
	if err := svc.MoveDepartment("ENG", "NOPE"); err == nil {
		t.Fatal("未知上级应报错")
	}
}

func TestCountAfterMove(t *testing.T) {
	svc := setup(t)
	if err := svc.AddAssignment("A1", "E1", "P1", "2026-01-01", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddAssignment("A2", "E2", "P3", "2026-01-01", ""); err != nil {
		t.Fatal(err)
	}
	// 迁移前：BE 是 ENG 后代，ENG 递归计数为 2。
	n, _ := svc.DeptCount("ENG", "2026-02-01", true)
	if n != 2 {
		t.Fatalf("迁移前 ENG 递归人数应为2, got %d", n)
	}
	// 把 BE 迁移为 ROOT 直属，ENG 递归只剩 FE 的 1 人；路径按当前树解析。
	if err := svc.MoveDepartment("BE", "ROOT"); err != nil {
		t.Fatal(err)
	}
	n, _ = svc.DeptCount("ENG", "2026-02-01", true)
	if n != 1 {
		t.Fatalf("迁移后 ENG 递归人数应为1, got %d", n)
	}
	n, _ = svc.DeptCount("ROOT", "2026-02-01", true)
	if n != 2 {
		t.Fatalf("ROOT 递归人数应仍为2, got %d", n)
	}
	rows, _ := svc.QueryAt("2026-02-01")
	for _, r := range rows {
		if r.EmployeeID == "E1" {
			if got := r.Current.Path[1]; got != "BE" {
				t.Fatalf("历史任职路径应按当前树解析, path=%v", r.Current.Path)
			}
		}
	}
}

func TestUnknownRefsAndBadDates(t *testing.T) {
	svc := setup(t)
	if err := svc.AddDepartment("X", "x", "NOPE"); err == nil {
		t.Fatal("未知上级应报错")
	}
	if err := svc.AddPost("X", "x", "NOPE"); err == nil {
		t.Fatal("未知部门应报错")
	}
	if err := svc.AddAssignment("X", "NOPE", "P1", "2026-01-01", ""); err == nil {
		t.Fatal("未知员工应报错")
	}
	if err := svc.AddAssignment("X", "E1", "NOPE", "2026-01-01", ""); err == nil {
		t.Fatal("未知岗位应报错")
	}
	if err := svc.AddAssignment("X", "E1", "P1", "2026-02-30", ""); err == nil {
		t.Fatal("非法日期应报错")
	}
	if err := svc.AddAssignment("X", "E1", "P1", "2026-03-01", "2026-03-01"); err == nil {
		t.Fatal("结束日不晚于开始日应报错")
	}
	if _, err := svc.QueryAt("2026-13-01"); err == nil {
		t.Fatal("非法查询日期应报错")
	}
	// 无任职员工在时点查询中保留，current 为空。
	rows, err := svc.QueryAt("2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Current != nil {
			t.Fatalf("无任何任职时 current 应为空, got %+v", r)
		}
	}
}

func TestPersistenceAcrossReload(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	mustInit(t, svc)
	if err := svc.AddAssignment("A1", "E1", "P1", "2026-01-01", "2026-06-01"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.state.RootDeptID != "ROOT" {
		t.Fatal("重开后根部门丢失")
	}
	if _, err := reloaded.Transfer("A1", "A2", "P2", "2026-03-01"); err != nil {
		t.Fatalf("重开后应能继续调岗: %v", err)
	}
}
