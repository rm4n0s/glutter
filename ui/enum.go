package ui

import "strconv"

// enumName returns names[v], or "<typeName>(v)" when v is out of range.
func enumName(names []string, v int, typeName string) string {
	if v >= 0 && v < len(names) {
		return names[v]
	}
	return typeName + "(" + strconv.Itoa(v) + ")"
}
