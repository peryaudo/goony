package winny

import (
	"encoding/hex"
	"strings"
)

func (k *FileKey) Match(keyword string) bool {
	for _, word := range strings.Split(keyword, " 　") {
		if len(word) == 0 {
			continue
		}

		if word[0] == '%' {
			if !k.matchHash(word) {
				return false
			}
			continue
		}

		if !k.matchSingle(word) {
			return false
		}
	}

	return true
}

func (k *FileKey) matchSingle(word string) bool {
	negate := false
	if word[0] == '-' {
		negate = true
		word = word[1:]
	}

	contain := strings.Contains(k.FileName, word)

	if negate {
		return !contain
	} else {
		return contain
	}
}

func (k *FileKey) matchHash(hash string) bool {
	b, err := hex.DecodeString(hash[1:])
	if err != nil {
		return false
	}
	if len(b) != len(k.Hash) {
		return false
	}

	for i := 0; i < len(k.Hash); i++ {
		if b[i] != k.Hash[i] {
			return false
		}
	}

	return true
}
