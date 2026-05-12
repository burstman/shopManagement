package dashboard

func strVal(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func firstChar(s *string) string {
	if s != nil && len(*s) > 0 {
		return string((*s)[0])
	}
	return "?"
}
