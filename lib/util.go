package lib

// Contains searches for a needle in a haystack
func Contains(haystack []string, needle string) bool {
	for _, value := range haystack {
		if value == needle {
			return true
		}
	}
	return false
}

// Dedup removes duplicates from a slice of strings
func Dedup(stringSlice []string) []string {
	var returnSlice []string
	for _, value := range stringSlice {
		if !Contains(returnSlice, value) {
			returnSlice = append(returnSlice, value)
		}
	}
	return returnSlice
}
