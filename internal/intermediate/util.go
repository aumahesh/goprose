package intermediate

func StringValue(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

func Int64Value(n *int64) int64 {
	if n == nil {
		return 0
	}
	return *n
}
