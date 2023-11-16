//go:build ((tinygo.wasm && wasi) || wasip1) && !nofastlyhostcalls

package main

import (
	"testing"

	"github.com/fastly/compute-sdk-go/secretstore"
)

func TestSecretStore(t *testing.T) {
	st, err := secretstore.Open("phrases")
	if err != nil {
		t.Fatal(err)
	}

	s, err := st.Get("my_phrase")
	if err != nil {
		t.Fatal(err)
	}

	v, err := s.Plaintext()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(v), "sssh! don't tell anyone!"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSecretFromBytes(t *testing.T) {
	const plaintext = "not a real secret"

	s, err := secretstore.SecretFromBytes([]byte(plaintext))
	if err != nil {
		t.Fatal(err)
	}

	v, err := s.Plaintext()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(v), plaintext; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
