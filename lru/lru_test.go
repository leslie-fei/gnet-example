package lru

import (
	"fmt"
	"os"
	"testing"
)

func TestLRUCacheOperations(t *testing.T) {
	filename := "test_cache.data"
	fmt.Printf("Creating LRU Cache with filename: %s\n", filename)
	cache, err := NewLRUCache(filename, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer func() {
		fmt.Println("Closing cache and removing file.")
		cache.Close()
		os.Remove(filename)
	}()

	// Set and Get
	fmt.Println("Setting key1 with value 'value1'.")
	err = cache.Set("key1", "value1")
	assertNoError(t, err, "Set")
	fmt.Println("Getting key1 to verify value.")
	value, found := cache.Get("key1")
	assertCorrectValue(t, value, "value1", found, "Get")

	// Update and Verify
	fmt.Println("Updating key1 with new value 'value2'.")
	err = cache.Set("key1", "value2")
	assertNoError(t, err, "Update")
	value, found = cache.Get("key1")
	assertCorrectValue(t, value, "value2", found, "Update Get")

	// Delete and Verify
	fmt.Println("Deleting key1 and verifying deletion.")
	err = cache.Delete("key1")
	assertNoError(t, err, "Delete")
	_, found = cache.Get("key1")
	if found {
		t.Errorf("Expected 'key1' to be deleted")
	}
}

func assertNoError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Errorf("Failed to %s: %v", msg, err)
	}
}

func assertCorrectValue(t *testing.T, got, want string, found bool, msg string) {
	if !found || got != want {
		t.Errorf("Expected to get '%s' for %s, got '%s'", want, msg, got)
	}
}

func TestUpdateWithDifferentLengths(t *testing.T) {
	fmt.Println("Testing update with different value lengths.")
	cache, _ := NewLRUCache("test.data", 1024)
	defer func() {
		fmt.Println("Closing cache and removing test data file.")
		cache.Close()
		os.Remove("test.data")
	}()

	fmt.Println("Setting key with a short value.")
	cache.Set("key", "short")
	fmt.Println("Updating key with a longer value.")
	cache.Set("key", "a much longer value than before")
	value, found := cache.Get("key")
	if !found || value != "a much longer value than before" {
		t.Errorf("Update failed, got: %s", value)
	}
}
