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

func FuzzParseGeoJSON(f *testing.F) {
	matches, err := filepath.Glob("testdata/*.json")
	if err != nil {
		f.Fatal(err)
	}

	for _, file := range matches {
		data, err := os.ReadFile(file)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		g, err := parseGeoJSON(data)

		var g2 Geo
		err2 := json.Unmarshal(data, &g2)

		if err != nil && err2 != nil {
			return
		}

		if err != nil && err2 == nil {
			// parseGeoJSON failed to parse "valid" json, but that's expected because it only knows about a limited subset
			return
		}

		if err == nil && err2 != nil {
			t.Errorf("parseGeoJSON parsed invalid json blob")
		}

		if err == nil && err2 == nil {
			if *g != g2 {
				t.Errorf("fuzzer found parsing mismatch: %#v != %#v", *g, g2)
			}
		}
	})
}
