package hostsfile

func itemInSliceString(item string, list []string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}

func itemInSliceInt(item int, list []int) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}

func removeFromSliceString(s string, slice []string) []string {
	pos := findPositionInSliceString(s, slice)
	for pos > -1 {
		slice = append(slice[:pos], slice[pos+1:]...)
		pos = findPositionInSliceString(s, slice)
	}
	return slice
}

func findPositionInSliceString(s string, slice []string) int {
	for index, v := range slice {
		if v == s {
			return index
		}
	}
	return -1
}

func removeFromSliceInt(s int, slice []int) []int {
	pos := findPositionInSliceInt(s, slice)
	for pos > -1 {
		slice = append(slice[:pos], slice[pos+1:]...)
		pos = findPositionInSliceInt(s, slice)
	}
	return slice
}

func removeOneFromSliceInt(s int, slice []int) []int {
	pos := findPositionInSliceInt(s, slice)
	if pos > -1 {
		slice = append(slice[:pos], slice[pos+1:]...)
	}
	return slice
}

func findPositionInSliceInt(s int, slice []int) int {
	for index, v := range slice {
		if v == s {
			return index
		}
	}
	return -1
}
