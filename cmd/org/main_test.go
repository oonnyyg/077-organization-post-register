package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLineProtocol(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("go", "run", ".", "--state", dir)
	cmd.Dir = filepath.Join("..", "..", "cmd", "org")
	stdin := strings.NewReader(strings.Join([]string{
		`{"cmd":"init","id":"ROOT","name":"根"}`,
		`{"cmd":"post_add","id":"P1","name":"岗","dept_id":"NOPE"}`,
		`{"cmd":"dept_add","id":"D1","name":"部门","parent_id":"ROOT"}`,
		`{"cmd":"post_add","id":"P1","name":"岗","dept_id":"D1"}`,
		`{"cmd":"employee_add","id":"E1","name":"张三"}`,
		`{"cmd":"assignment_add","id":"A1","employee_id":"E1","post_id":"P1","start":"2026-01-01","end":""}`,
		`{"cmd":"query_at","date":"2026-01-01"}`,
		`{"cmd":"query_at","date":"bad"}`,
		`{"cmd":"nope"}`,
	}, "\n") + "\n")
	cmd.Stdin = stdin
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run: %v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 9 {
		t.Fatalf("期望9行响应, got %d: %s", len(lines), out)
	}
	if !strings.Contains(lines[1], `"ok":false`) {
		t.Fatalf("未知部门应报错: %s", lines[1])
	}
	if !strings.Contains(lines[6], `"D1"`) || !strings.Contains(lines[6], `"张三"`) {
		t.Fatalf("时点查询结果异常: %s", lines[6])
	}
	if !strings.Contains(lines[7], `"ok":false`) || !strings.Contains(lines[8], `"ok":false`) {
		t.Fatal("非法日期与未知命令应返回失败JSON且继续处理")
	}
}
