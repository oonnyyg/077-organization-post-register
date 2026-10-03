package org

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const stateFile = "state.json"

// Load 从状态目录读取数据；目录或文件不存在时返回空状态。
func Load(dir string) (*State, error) {
	path := filepath.Join(dir, stateFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return newState(), nil
	}
	if err != nil {
		return nil, err
	}
	st := newState()
	if err := json.Unmarshal(data, st); err != nil {
		return nil, fmt.Errorf("状态文件损坏: %w", err)
	}
	if st.Departments == nil {
		st.Departments = map[string]*Department{}
	}
	if st.Posts == nil {
		st.Posts = map[string]*Post{}
	}
	if st.Employees == nil {
		st.Employees = map[string]*Employee{}
	}
	return st, nil
}

// save 原子写：先写临时文件再重命名，避免写入中途失败破坏旧状态。
func (s *State) save(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, stateFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
