package handler

// contains 检查 slice 是否包含元素（工具函数）
func contains(slice []uint, item uint) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func intersectUintSlices(left, right []uint) []uint {
	if len(left) == 0 || len(right) == 0 {
		return []uint{}
	}

	allowed := make(map[uint]struct{}, len(right))
	for _, value := range right {
		allowed[value] = struct{}{}
	}

	result := make([]uint, 0, len(left))
	seen := make(map[uint]struct{}, len(left))
	for _, value := range left {
		if _, ok := allowed[value]; !ok {
			continue
		}
		if _, duplicated := seen[value]; duplicated {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
