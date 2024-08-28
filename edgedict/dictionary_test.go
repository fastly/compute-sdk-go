// Copyright 2022 Fastly, Inc.

package edgedict

import "testing"

func TestDictionary(t *testing.T) {
	var d *Dictionary
	val, err := d.Get("xyzzy")
	if err != ErrKeyNotFound {
		t.Errorf("Expected get on nil dictionary to return ErrKeyNotFound")
	}
	// check val despite err being non-nil
	if val != "" {
		t.Errorf("Expected get on nil dictionary to return empty string")
	}
}
