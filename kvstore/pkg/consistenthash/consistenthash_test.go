package consistenthash

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashing(t *testing.T) {
	hash := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	// Add nodes to the hash
	hash.Add("6", "4", "2")

	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCases {
		assert.Equal(t, v, hash.Get(k), "Incorrect node returned for key "+k)
	}

	// Add a new node
	hash.Add("8")

	// Keys that were previously mapped to 2 should now map to 8
	assert.Equal(t, "8", hash.Get("27"), "Key should be remapped to the new node")

	// Remove a node
	hash.Remove("8")
	assert.Equal(t, "2", hash.Get("27"), "Key should be remapped back after node removal")
}

func TestGetAll(t *testing.T) {
	hash := New(3, nil)
	hash.Add("6", "4", "2")

	all := hash.GetAll()
	assert.Len(t, all, 3, "Expected 3 unique nodes")
	assert.Contains(t, all, "6", "Missing node 6")
	assert.Contains(t, all, "4", "Missing node 4")
	assert.Contains(t, all, "2", "Missing node 2")
}

func TestEmptyHash(t *testing.T) {
	hash := New(3, nil)
	assert.Equal(t, "", hash.Get("key"), "Expected empty result for empty hash")
}