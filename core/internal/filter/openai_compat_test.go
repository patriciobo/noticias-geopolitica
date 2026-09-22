package filter

import "testing"

func TestExtractJSONObject(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain json", `{"a":1}`, `{"a":1}`},
		{"markdown fences", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"bare fences", "```\n{\"a\":1}\n```", `{"a":1}`},
		{"leading prose", `Claro, acá está: {"a":1}`, `{"a":1}`},
		{"trailing prose", `{"a":1} espero que ayude`, `{"a":1}`},
		{"no braces", "sin json acá", "sin json acá"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractJSONObject(c.in); got != c.want {
				t.Errorf("extractJSONObject(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
