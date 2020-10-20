package trie

import (
	"testing"

	"github.com/ledgerwatch/turbo-geth/common"
)

func TestValue(t *testing.T) {
	t.Skip("should be restored. skipped for turbo-geth")

	h := newHasher(false)
	var hn common.Hash
	h.hash(valueNode([]byte("BLAH")), false, hn[:])
	expected := "0x0"
	if common.ToHex(hn[:]) != expected {
		t.Errorf("Expected %s, got %s", expected, common.ToHex(hn[:]))
	}
}