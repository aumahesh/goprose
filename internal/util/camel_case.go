package util

import (
	"strings"
)

func ToCamelCase(s string) string {
	x := strings.TrimSpace(s)
	xs := strings.Split(x, "_")
	cs := []string{}
	for _, y := range xs {
		capy := strings.ToUpper(string(y[0]))
		cy := capy + string(y[1:])
		cs = append(cs, cy)
	}
	return strings.Join(cs, "")
}
