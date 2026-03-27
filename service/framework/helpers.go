package framework

func boolToInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}
