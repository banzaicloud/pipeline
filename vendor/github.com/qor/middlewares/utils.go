package middlewares

// getRIndex get right index from string slice
func getRIndex(strs []string, str string) (int, bool) {
	for i := len(strs) - 1; i >= 0; i-- {
		if strs[i] == str {
			return i, true
		}
	}
	return -1, false
}

func uniqueAppend(strs []string, str string) []string {
	for _, s := range strs {
		if s == str {
			return strs
		}
	}

	return append(strs, str)
}
