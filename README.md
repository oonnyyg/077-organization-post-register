# 组织架构与员工主岗管理 (organization-post-register)

仅依赖 Go 标准库的命令行服务：通过 stdin/stdout 逐行交换 JSON，
管理单根部门树、岗位、员工、主岗任职（含调岗），并支持时点归属、
调岗历史和部门人数查询。状态持久化在 `--state` 指定目录的
`state.json` 中，正常退出后重开自动保留数据。

## 运行

```sh
go run ./cmd/org --state ./data
```

每一行 stdin 是一个 JSON 请求，每一行 stdout 是一个 JSON 响应。
EOF 即正常退出；单条命令报错不影响后续命令。

成功响应形如 `{"ok":true,...}`；失败响应形如
`{"ok":false,"error":"..."}`，失败时状态不变。

## 命令协议

所有 ID 在各自类别（部门 / 岗位 / 员工）内唯一且非空。
日期均为真实日历日期，格式 `YYYY-MM-DD`。
任职区间为半开区间 `[start, end)`：开始日包含、结束日不包含；
`end` 为空字符串表示无限期，且 end 必须严格晚于 start。

| cmd | 关键字段 | 说明 |
| --- | --- | --- |
| `init_root` | `id`, `name` | 指定不可迁移的根部门，仅允许执行一次 |
| `dept_add` | `id`, `name`, `parent_id` | 在已有上级下登记部门 |
| `dept_move` | `id`, `new_parent_id` | 调整上级；不能引用自身或后代，根不能迁移 |
| `post_add` | `id`, `name`, `dept_id` | 岗位固定归属一个已有部门 |
| `employee_add` | `id`, `name` | 登记员工；员工可以始终没有任职 |
| `assignment_add` | `employee_id`, `post_id`, `start`, `end` | 录入主岗任职；同一员工区间不可重叠，首尾相接允许；同一岗位可多人任职。返回自动生成的任职 ID（`a1`、`a2` …） |
| `transfer` | `id`, `new_post_id`, `date` | 在 `date` 结束原任职并同日起任新岗位；`date` 须严格晚于原开始日且严格早于原结束日，新任职沿用原结束日 |
| `history` | `employee_id` | 返回该员工全部任职，按开始日排序 |
| `query_date` | `date` | 返回每名员工当日主岗、直接部门及根到部门的完整路径；无任职者保留且 `assignment` 为空 |
| `dept_count` | `dept_id`, `date`, `recursive` | 统计当日部门人数；`recursive=true` 包含全部后代部门，每名员工只计一次 |

### 调岗原子性

`transfer` 先校验新岗位存在、日期合法、新区间不冲突，
全部通过后才同时写入两条记录（原任职结束日改为 `date`，
新任职为 `[date, 原结束日)`）。任一校验失败时，
原任职不会被提前截断，也不会产生半截新任职。

### 时点查询与部门路径

`query_date` 与 `dept_count` 一律按**当前**部门树解析部门与路径，
历史任职不保存旧组织树快照。部门迁移后，查询历史日期时
该员工归属路径也会反映迁移后的结构。

`query_date` 响应中每个员工包含：

```json
{
  "id": "e1",
  "name": "Alice",
  "assignment": {
    "assignment_id": "a2",
    "post_id": "p2",
    "post_name": "Lead",
    "dept_id": "eng",
    "dept_name": "Engineering",
    "path": ["root", "eng"],
    "path_names": ["HQ", "Engineering"]
  }
}
```

当日无任职时 `assignment` 字段缺省（`omitempty`）。

## 示例

```sh
go run ./cmd/org --state ./data <<'EOF'
{"cmd":"init_root","id":"root","name":"HQ"}
{"cmd":"dept_add","id":"eng","name":"Engineering","parent_id":"root"}
{"cmd":"post_add","id":"p1","name":"Dev","dept_id":"eng"}
{"cmd":"employee_add","id":"e1","name":"Alice"}
{"cmd":"assignment_add","employee_id":"e1","post_id":"p1","start":"2026-01-01","end":"2026-06-01"}
{"cmd":"query_date","date":"2026-03-01"}
EOF
```

## 测试

```sh
go test ./...
```

测试覆盖：相邻（首尾相接）任职、任职重叠与非法日期拒绝、
调岗成功沿用原结束日、调岗失败时原任职不被截断、
部门迁移后时点路径与人数统计变化、以及退出重开后的数据保留
（含一个通过 `go run` 驱动的 CLI 端到端测试）。

## 持久化

每次成功的变更都会把完整状态写入
`<state>/state.json.tmp` 后原子重命名为 `state.json`，
避免崩溃留下截断文件。
