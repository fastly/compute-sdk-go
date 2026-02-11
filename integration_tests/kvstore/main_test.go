//go:build wasip1 && !nofastlyhostcalls

// Copyright 2023 Fastly, Inc.

package main

import (
	"maps"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/fastly/compute-sdk-go/kvstore"
)

func TestKVStore(t *testing.T) {
	store, err := kvstore.Open("example-test-kv-store")
	if err != nil {
		t.Fatal(err)
	}

	hello, err := store.Lookup("hello")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := hello.String(), "world"; got != want {
		t.Errorf("Lookup: got %q, want %q", got, want)
	}

	_, err = store.Lookup("animal")
	if err == nil {
		t.Error("expected Lookup failure before insert")
	}

	err = store.Insert("animal", strings.NewReader("cat"))
	if err != nil {
		t.Fatal(err)
	}

	animal, err := store.Lookup("animal")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := animal.String(), "cat"; got != want {
		t.Errorf("Insert: got %q, want %q", got, want)
	}

	if err = store.Delete("animal"); err != nil {
		t.Fatal(err)
	}

	_, err = store.Lookup("animal")
	if err == nil {
		t.Error("expected Lookup failure after delete")
	}

	/*
		// TODO(athomason) address inconsistent behavior in viceroy and production
		if err = store.Delete("nonexistent"); err != nil {
			t.Fatal(err)
		}
	*/

	wantListKeys := make(map[string]bool)
	for i := 0; i < 3000; i++ {
		s := strconv.Itoa(i)
		store.Insert(s, strings.NewReader(s))
		if strings.HasPrefix(s, "20") {
			wantListKeys[s] = true
		}
	}

	// iterate over the keys
	it := store.List(&kvstore.ListConfig{Mode: kvstore.ListConsistencyEventual, Limit: 20, Prefix: "20"})
	gotListKeys := make(map[string]bool)
	var pageCount int
	var keysCount int
	for it.Next() {
		pageCount++
		page := it.Page()
		keysCount += len(page.Data)
		for _, k := range page.Data {
			gotListKeys[k] = true
		}
	}

	if err := it.Err(); err != nil {
		t.Error("error during iteration:", err)
	}

	if pageCount != 6 || keysCount != 111 {
		t.Error("Expected page/keys count wrong: want page 6, keys 111 got page ", pageCount, ", keys", keysCount)
	}

	if len(gotListKeys) != 111 {
		t.Error("Expected list keys count wrong: want", len(wantListKeys), "got", len(gotListKeys))
	}

	if !maps.Equal(wantListKeys, gotListKeys) {
		t.Errorf("Expected got/want keys mismatch: want=%v, got=%v", mapKeys(wantListKeys), mapKeys(gotListKeys))
	}
}

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
