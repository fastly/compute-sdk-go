//go:build !tinygo.wasm && !wasi
// +build !tinygo.wasm,!wasi

// Copyright 2022 Fastly, Inc.

package geo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseEdgeCases(t *testing.T) {
	t.Parallel()

	for _, testcase := range []struct {
		name  string
		input string
		want  Geo
		err   bool
	}{
		{
			name:  "empty",
			input: ``,
			want:  Geo{},
		},
		{
			name:  "open brace",
			input: `{`,
			err:   true,
		},
		{
			name:  "close brace",
			input: `}`,
			err:   true,
		},
		{
			name:  "spaces",
			input: `     `,
			err:   true,
		},
		{
			name:  "empty object",
			input: `{}`,
			want:  Geo{},
		},
		{
			name:  "empty array",
			input: `[]`,
			err:   true,
		},
		{
			name:  "zero byte",
			input: string([]byte{0}),
			err:   true,
		},
		{
			name:  "emoji",
			input: `{"as_name": "😎 Networks"}`,
			want:  Geo{AsName: "😎 Networks"},
		},
		{
			name:  "lots of whitespace",
			input: `{"as_name": "Foo",	` + "\n\n" + `			  "metro_code": 92       }     `,
			want:  Geo{AsName: "Foo", MetroCode: 92},
		},
		{
			name:  "key is not a string",
			input: `{92: "metro_code"}`,
			err:   true,
		},
		// TODO
		//{
		//	name:  "too many colons",
		//	input: `{"as_name":: "Foo", "metro_code": 92}`,
		//	err:   true,
		//},
		//{
		//	name:  "too many commas",
		//	input: `{"as_name": "Foo",, "metro_code": 92}`,
		//	err:   true,
		//},
		//{
		//	name:  "wrong delimiter",
		//	input: `{"as_name", "Foo": "metro_code": 92}`,
		//	err:   true,
		//},
		//{
		//	name:  "misplaced comma",
		//	input: `{, "as_name": "foo"}`,
		//	err:   true,
		//},
		{
			name:  "unknown key array",
			input: `{"as_name": "Foo", "something_else": [[[]]], "metro_code": 92}`,
			want:  Geo{AsName: "Foo", MetroCode: 92},
		},
		{
			name:  "unknown key object",
			input: `{"as_name": "Foo", "something_else": {"foo": "bar"}, "metro_code": 92}`,
			want:  Geo{AsName: "Foo", MetroCode: 92},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			g, err := parseGeoJSON([]byte(testcase.input))
			if testcase.err && err == nil {
				t.Errorf("want error, have none")
			}
			if !testcase.err && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !testcase.err && g != nil && *g != testcase.want {
				t.Errorf("want: %#+v", testcase.want)
				t.Errorf("have: %#+v", *g)
			}
		})
	}
}

func TestParseTestdata(t *testing.T) {
	t.Parallel()

	matches, err := filepath.Glob("testdata/*.json")
	if err != nil {
		t.Fatal(err)
	}

	for _, filename := range matches {
		t.Run(filepath.Base(filename), func(t *testing.T) {
			buf, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}

			g1, err := parseGeoJSON(buf)
			if err != nil {
				t.Fatal(err)
			}

			var g2 Geo
			if err := json.Unmarshal(buf, &g2); err != nil {
				t.Fatal(err)
			}

			if *g1 != g2 {
				t.Errorf("parse:          %#+v", *g1)
				t.Errorf("json.Unmarshal: %#+v", g2)
			}
		})
	}
}

func TestParseFuzz(t *testing.T) {
	t.Parallel()

	t.Skip("TODO")
}
