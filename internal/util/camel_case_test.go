package util

import (
	"testing"
)

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{s: "hello", want: "Hello"},
		{s: "helloworld", want: "Helloworld"},
		{s: "hello_world", want: "HelloWorld"},
		{s: " hello ", want: "Hello"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := ToCamelCase(tt.s); got != tt.want {
				t.Errorf("ToCamelCase() = %v, want %v", got, tt.want)
			}
		})
	}
}
