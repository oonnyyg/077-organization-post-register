# 组织架构与员工主岗登记

仅依赖 Go 标准库的命令行服务：通过 stdin/stdout 逐行交换 JSON，维护部门树、
岗位、员工、主岗任职记录，支持调岗、时点查询、部门人数统计与任职历史，
数据持久化到 `--state` 指定目录（`state.json`，临时文件 + 重命名原子写入），
正常退出后重开数据保留。

## 运行

```
go run ./cmd/org --state ./状态目录
```

每行进一个 JSON 请求，每行出一个 JSON 响应：

```json
{"ok": true, "result": {"status": "ok"}}
{"ok": false, "error": "员工 \"NOPE\" 不存在"}
```

业务错误不影响进程：输出 `{"ok":false,...}` 后继续处理下一行，且失败命令不改动状态。

## 命令

| cmd | 字段 | 说明 |
| --- | --- | --- |
| `init` | `id`,`name` | 指定根部门，只能一次；根部门不可迁移 |
| `dept_add` | `id`,`name`,`parent_id` | 新增部门，上级必须已存在 |
| `dept_move` | `id`,`new_parent_id` | 调整上级；根不可迁，不能引用自身或后代 |
| `post_add` | `id`,`name`,`dept_id` | 岗位固定归属一个已有部门 |
| `employee_add` | `id`,`name` | 登记员工 |
| `assignment_add` | `id`,`employee_id`,`post_id`,`start`,`end` | 录入任职；`end` 可省略/空串 |
| `assignment_end` | `id`,`date` | 在指定日期提前结束任职 |
| `transfer` | `id`,`new_assignment_id`,`new_post_id`,`date` | 当日结束旧任职并开启新任职 |
| `query_at` | `date` | 查询某日全体员工主岗与根→部门路径 |
| `dept_count` | `dept_id`,`date`,`recursive` | 部门当日人数，`recursive` 含全部后代 |
| `history` | `employee_id` | 员工全部任职，按开始日排序 |

各类编号（部门/岗位/员工/任职）在各自命名空间内唯一且非空。

## 时间规则

- 日期均为 `YYYY-MM-DD` 真实日期（如 `2026-02-30` 报错）。
- 任职区间为半开 `[start, end)`：开始日包含、结束日不包含；空 `end` 为无限期。
- `end` 必须严格晚于 `start`。
- 同一员工的主岗任职时间区间不能重叠；首尾相接（上一段 `end` = 下一段 `start`）允许。
- 同一岗位可由多名员工同时任职。
- `assignment_end` / `transfer` 的日期必须严格晚于该条开始日，且早于已有结束日。
- `transfer` 新任职沿用原结束日；先校验新岗位存在、日期合法、与同员工其他任职
  无冲突，再同时写入两条记录。任何校验失败都不会提前截断原任职，也不产生新记录。

## 查询语义

- `query_at` 返回所有员工（包括无任职者，`current` 为 `null`）。命中任职时返回
  任职编号、岗位、直接部门，以及按**当前**部门树解析的根→部门编号/名称路径；
  历史任职不保存旧组织树，部门迁移后所有查询路径即时变化。
- `dept_count`：`recursive:false` 只统计本部门当日在岗员工；`true` 时包含全部
  后代部门。每名员工当日最多一条主岗，因此只计一次。

## 测试

```
go test ./...
```

覆盖：相邻（首尾相接）任职、重叠拒绝、调岗成功与失败不截断、部门迁移规则、
迁移前后部门人数与路径变化、未知引用/非法日期、跨进程持久化、逐行协议。
