package checksum

import (
	"crypto"
	"strings"
	"testing"
)

func TestHashFromIdentifier(t *testing.T) {
	tests := []struct {
		id       string
		expected Hash
	}{
		{id: "md4", expected: Hash{crypto.MD4}},
		{id: "sha512", expected: Hash{crypto.SHA512}},
	}

	for _, tt := range tests {
		got, err := FromIdentifier(tt.id)
		assertNoErr(t, err)
		assertEqual(t, got, tt.expected)
	}
}

func TestHashFromIdentifierUnknown(t *testing.T) {
	got, err := FromIdentifier("foobar")
	assertErr(t, err)
	assertEqual(t, got, Hash{})
}

func TestHashToIdentifier(t *testing.T) {
	tests := []struct {
		hash     Hash
		expected string
	}{
		{hash: Hash{crypto.MD4}, expected: "md4"},
		{hash: Hash{crypto.SHA512}, expected: "sha512"},
		{hash: Hash{crypto.SHA3_384}, expected: "sha3_384"},
	}

	for _, tt := range tests {
		got, err := tt.hash.ToIdentifier()
		assertNoErr(t, err)
		assertEqual(t, got, tt.expected)
	}
}

func TestHashToIdentifierUnsupported(t *testing.T) {
	tests := []struct{ hash Hash }{
		{hash: Hash{crypto.SHA224}},
		{hash: Hash{crypto.BLAKE2b_256}},
		{hash: Hash{crypto.RIPEMD160}},
	}

	for _, tt := range tests {
		got, err := tt.hash.ToIdentifier()
		assertErr(t, err)
		if !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("expected an error containing unsupported, got '%v'", err)
		}
		assertEqual(t, got, "")
	}
}
