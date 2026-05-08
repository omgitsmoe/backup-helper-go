package checksum

import (
	"crypto"
	"fmt"
)

type Hash struct {
	crypto.Hash
}

func (h Hash) ToIdentifier() string {
	switch h.Hash {
	case crypto.MD4:
		return "md4"
	case crypto.MD5:
		return "md5"
	case crypto.SHA1:
		return "sha1"
	case crypto.SHA256:
		return "sha256"
	case crypto.SHA384:
		return "sha384"
	case crypto.SHA3_224:
		return "sha3_224"
	case crypto.SHA3_256:
		return "sha3_256"
	case crypto.SHA3_384:
		return "sha3_384"
	case crypto.SHA3_512:
		return "sha3_512"
	case crypto.SHA512:
		return "sha512"

	case crypto.SHA224:
		fallthrough
	case crypto.SHA512_224:
		fallthrough
	case crypto.SHA512_256:
		fallthrough
	case crypto.BLAKE2b_256:
		fallthrough
	case crypto.BLAKE2b_384:
		fallthrough
	case crypto.BLAKE2b_512:
		fallthrough
	case crypto.BLAKE2s_256:
		fallthrough
	case crypto.MD5SHA1:
		fallthrough
	case crypto.RIPEMD160:
		return "unsupported hash " + h.Hash.String()
	}

	panic(fmt.Sprintf("unexpected crypto.Hash: %#v", h.Hash))
}

var identifierToHash = map[string]Hash{
    "md4":      {crypto.MD4},
    "md5":      {crypto.MD5},
    "sha1":     {crypto.SHA1},
    "sha256":   {crypto.SHA256},
    "sha384":   {crypto.SHA384},
    "sha3_224": {crypto.SHA3_224},
    "sha3_256": {crypto.SHA3_256},
    "sha3_384": {crypto.SHA3_384},
    "sha3_512": {crypto.SHA3_512},
    "sha512":   {crypto.SHA512},
}

func FromIdentifier(id string) (Hash, error) {
    if h, ok := identifierToHash[id]; ok {
        return h, nil
    }
    return Hash{}, fmt.Errorf("unknown hash identifier: %q", id)
}
