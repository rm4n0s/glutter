package widgets

// enumName is the Dart name of an enum value whose zero value is "unspecified"
// and whose other values, in Flutter's declaration order, are 1..len(names).
func enumName(v int, names ...string) string {
	if v >= 1 && v <= len(names) {
		return names[v-1]
	}
	return "unspecified"
}
