// Copyright 2022 Fastly, Inc.

package configstore

import "testing"

func TestStore(t *testing.T) {
	var c *Store
	val, err := c.Get("xyzzy")
	if err != ErrKeyNotFound {
		t.Errorf("Expected get on nil configstore to return ErrKeyNotFound")
	}
	// check val despite err being non-nil
	if val != "" {
		t.Errorf("Expected get on nil configstore to return empty string")
	}
}
