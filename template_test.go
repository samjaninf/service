// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

package service

import (
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	funcs := map[string]tmplFunc{
		"upper": func(s string) (string, error) { return strings.ToUpper(s), nil },
		"quote": func(s string) (string, error) { return `"` + s + `"`, nil },
	}
	cases := []struct {
		name string
		tmpl string
		data map[string]any
		want string
	}{
		{
			name: "value and pipe",
			tmpl: "hi {{Name | upper}} {{Name | quote | upper}}",
			data: map[string]any{"Name": "bob"},
			want: `hi BOB "BOB"`,
		},
		{
			name: "if true and false",
			tmpl: "{{if A}}a={{A}}{{end}}{{if B}}b={{B}}{{end}}",
			data: map[string]any{"A": "1", "B": ""},
			want: "a=1",
		},
		{
			name: "if else",
			tmpl: "{{if X}}yes{{else}}no{{end}}",
			data: map[string]any{"X": ""},
			want: "no",
		},
		{
			name: "range over list with dot",
			tmpl: "{{range Args}} {{. | quote}}{{end}}",
			data: map[string]any{"Args": []string{"a", "b c"}},
			want: ` "a" "b c"`,
		},
		{
			name: "range empty list is falsy",
			tmpl: "{{if Deps}}has{{else}}none{{end}}",
			data: map[string]any{"Deps": []string(nil)},
			want: "none",
		},
		{
			name: "nested range inside if",
			tmpl: "{{if Args}}[{{range Args}}{{.}},{{end}}]{{end}}",
			data: map[string]any{"Args": []string{"x", "y"}},
			want: "[x,y,]",
		},
		{
			name: "trim markers",
			tmpl: "a\n{{- if X}} b {{end -}}\nc",
			data: map[string]any{"X": "1"},
			want: "a b c",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := renderTemplate(tc.tmpl, tc.data, funcs)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseOnceRenderMany(t *testing.T) {
	parsed, err := parseTemplate("{{if Args}}[{{range Args}}{{.}};{{end}}]{{else}}none{{end}} {{Name}}")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		data map[string]any
		want string
	}{
		{map[string]any{"Args": []string{"a", "b"}, "Name": "x"}, "[a;b;] x"},
		{map[string]any{"Args": []string(nil), "Name": "y"}, "none y"},
		{map[string]any{"Args": []string{"z"}, "Name": "w"}, "[z;] w"},
	}
	for i := 0; i < 2; i++ { // render the same parsed template repeatedly
		for _, tc := range cases {
			got, err := parsed.render(tc.data, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("render got %q, want %q", got, tc.want)
			}
		}
	}
}

func TestRenderTemplateErrors(t *testing.T) {
	cases := []struct {
		name string
		tmpl string
		data map[string]any
	}{
		{"unclosed", "{{ Name ", map[string]any{"Name": "x"}},
		{"unknown key", "{{ Missing }}", map[string]any{}},
		{"unknown func", "{{ Name | nope }}", map[string]any{"Name": "x"}},
		{"unexpected end", "text {{end}}", map[string]any{}},
		{"range over string", "{{range Name}}{{.}}{{end}}", map[string]any{"Name": "x"}},
		{"string as list", "{{Args}}", map[string]any{"Args": []string{"a"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := renderTemplate(tc.tmpl, tc.data, nil); err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
