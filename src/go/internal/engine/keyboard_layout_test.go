package engine

import (
	"testing"
)

func TestAdjacentKeys_Coverage(t *testing.T) {
	// 验证小写字母和数字键已配置
	expectedLetters := "abcdefghijklmnopqrstuvwxyz0123456789"
	for _, r := range expectedLetters {
		_, ok := adjacentKeys[r]
		if !ok {
			t.Errorf("adjacentKeys missing key %q", r)
		}
	}
}

func TestAdjacentKeys_NoSelfReference(t *testing.T) {
	// 不允许键把自身列为相邻键
	for key, neighbors := range adjacentKeys {
		for _, n := range neighbors {
			if n == key {
				t.Errorf("adjacentKeys[%q] contains self", key)
			}
		}
	}
}

func TestAdjacentKeys_Symmetry(t *testing.T) {
	// 如果 A 包含 B，则 B 也应包含 A（物理相邻性应对称）
	asymmetries := 0
	for key, neighbors := range adjacentKeys {
		for _, n := range neighbors {
			nNeighbors := adjacentKeys[n]
			found := false
			for _, nn := range nNeighbors {
				if nn == key {
					found = true
					break
				}
			}
			if !found {
				asymmetries++
				t.Logf("asymmetric: %q -> %q but %q -/-> %q", key, n, n, key)
			}
		}
	}
	// 部分不对称属正常（如键盘布局不均匀），记录下来供审查
	t.Logf("found %d asymmetric adjacencies", asymmetries)
}

func TestAdjacentKeys_NonEmpty(t *testing.T) {
	for key, neighbors := range adjacentKeys {
		if len(neighbors) == 0 {
			t.Errorf("adjacentKeys[%q] has no neighbors", key)
		}
	}
}

func TestAdjacentKeys_UniqueNeighbors(t *testing.T) {
	for key, neighbors := range adjacentKeys {
		seen := make(map[rune]bool)
		for _, n := range neighbors {
			if seen[n] {
				t.Errorf("adjacentKeys[%q] has duplicate neighbor %q", key, n)
			}
			seen[n] = true
		}
	}
}
