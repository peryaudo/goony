package winny

import (
	"testing"
)

func TestCmdQuery(t *testing.T) {
	expecteds := [][]byte{
		[]byte{
			0x0, 0x0, 0x1, 0x0, 0x9b, 0x30, 0x1, 0x0, 0x21, 0x25, 0x65, 0x61, 0x64, 0x34, 0x31, 0x64,
			0x34, 0x37, 0x66, 0x61, 0x63, 0x38, 0x30, 0x39, 0x62, 0x39, 0x61, 0x66, 0x64, 0x36, 0x32,
			0x65, 0x66, 0x62, 0x30, 0x35, 0x31, 0x35, 0x37, 0x62, 0x32, 0x66, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0xc0, 0xa8, 0x0, 0x2, 0x28, 0x2d, 0x0, 0x0},

		// TODO(peryaudo): This example is ethically really really bad, so replace before publishing the source code.
		[]byte{
			0x0,                // IsReply
			0x1,                // IsSpread
			0x1,                // IsDownstream
			0x0,                // IsBbs
			0x4, 0x3, 0x0, 0x0, // Id
			0x0,                                                   // keywordLen
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // Trip [11]byte
			0x0,      // nodesLen
			0x1, 0x0, // keysLen

			// Begin queryKey struct
			0x76, 0x6a, 0x9c, 0x5f, 0x3f, 0x1e, // Node
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // BbsNode
			0x4, 0xb8, 0x4d, 0x23, // Size
			0x2a, 0x66, 0x62, 0x44, 0xda, 0x9c, 0x16, 0x2, // Hash
			0x72, 0x79, 0xfc, 0x3f, 0xaa, 0x46, 0x3c, 0x2c, // ...
			0x42,       // fileNameLen = 66
			0x1d, 0x1c, // checksum
			0xc8, 0xe0, 0x9d, 0x5d, 0x73, 0x1, 0x3e, 0xcd, 0x4f, 0x51, // FileName
			0xce, 0xab, 0xd6, 0xbd, 0x36, 0xf1, 0x7c, 0xaa, 0x9a, 0xe2, // ...
			0x6d, 0x2d, 0x89, 0x80, 0x7e, 0xeb, 0x6a, 0xdb, 0xf1, 0xee,
			0x57, 0x3e, 0x4, 0x43, 0xb6, 0xdb, 0x38, 0x32, 0xfd, 0x29,
			0xae, 0xf8, 0x99, 0xfc, 0x79, 0x8f, 0xc5, 0x6f, 0x34, 0x23,
			0x3c, 0x9d, 0x1b, 0xb0, 0x18, 0xb7, 0xa6, 0xc2, 0x15, 0x6b,
			0xd9, 0xd3, 0x1f, 0x92, 0xc1, 0xf2,
			0x6d, 0x37, 0x47, 0x67, 0x59, 0x6a, 0x68, 0x49, 0x69, 0x55, 0x0, // Trip
			0x0,       // bbsTripLen
			0x14, 0x2, // Ttl
			0xd5, 0x29, 0x1c, 0x0, // RefCnt
			0xd, 0x16, 0x2a, 0x55, // Timestamp
			0x0, // IsIgnored
			0x4, // KeyVer
		}}
	for _, expected := range expecteds {
		var q cmdQuery
		err := q.Unmarshal(expected)
		if err != nil {
			t.Error(err)
			return
		}
		actual, err := q.Marshal()
		if err != nil {
			t.Error(err)
			return
		}
		t.Logf("%#v", q)
		if len(actual) != len(expected) {
			t.Errorf("len mismatch %d vs %d expected %#v actual %#v", len(expected), len(actual), expected, actual)
			return
		}
		for i := 0; i < len(actual); i++ {
			if actual[i] != expected[i] {
				t.Errorf("byte mismatch expected %#v actual %#v", expected, actual)
				return
			}
		}
	}

}
func TestCmdAddr(t *testing.T) {
	expecteds := [][]byte{
		[]byte{
			0x99, 0xe0, 0xd1, 0x88, 0x7a, 0x24, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x78, 0x0, 0x0, 0x0, 0x6, 0x3, 0xa, 0x83, 0x41, 0x83, 0x6a, 0x83, 0x81,
			0x6d, 0x70, 0x34, 0x31, 0x38, 0x8b, 0xd6, 0x83, 0x51, 0x81, 0x5b, 0x83, 0x80}}
	for _, expected := range expecteds {
		var a cmdAddr
		err := a.Unmarshal(expected)
		if err != nil {
			t.Error(err)
			return
		}
		actual, err := a.Marshal()
		if err != nil {
			t.Error(err)
			return
		}
		if len(actual) != len(expected) {
			t.Errorf("len mismatch expected %#v actual %#v", expected, actual)
			return
		}
		for i := 0; i < len(actual); i++ {
			if actual[i] != expected[i] {
				t.Errorf("byte mismatch expected %#v actual %#v", expected, actual)
				return
			}
		}
	}
}

func TestCmdSpreadCond(t *testing.T) {
	expecteds := [][]byte{
		[]byte{0x83, 0x77, 0x83, 0x41, 0x20, 0x2d, 0x83, 0x41, 0x83, 0x6a, 0x83, 0x81, 0x20, 0x2d, 0x31, 0x38,
			0x8b, 0xd6, 0x83, 0x51, 0x81, 0x5b, 0x83, 0x80, 0x20, 0x2d, 0x93, 0xaf, 0x90, 0x6c, 0x20, 0x2d,
			0x8f, 0xac, 0x90, 0xe0, 0x20, 0x2d, 0x41, 0x4e, 0x49, 0x4d, 0x45, 0x20, 0x2d, 0x93, 0xc1, 0x8e,
			0x42, 0x0, 0x0, 0x38, 0x77, 0x15, 0x1, 0x0, 0xec, 0x12, 0x0, 0x8f, 0x4, 0xd2, 0x77, 0x0,
			0x30, 0xd0, 0x1, 0x24, 0xe9, 0x12, 0x0, 0x1b, 0x94, 0x41, 0x0, 0x30, 0xe9, 0x12, 0x0, 0x0,
			0x30, 0xd0, 0x1, 0x40, 0xe9, 0x12, 0x0, 0x87, 0x50, 0x4d, 0x0, 0x8, 0x71, 0xd0, 0x1, 0x34,
			0xb0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x78,
			0xea, 0x12, 0x0, 0x9, 0xa5, 0x4d, 0x0, 0x0, 0x0, 0x0, 0x0, 0x8, 0x71, 0xd0, 0x1, 0xa4,
			0xea, 0x12, 0x0, 0xb7, 0x52, 0x4d, 0x0, 0x34, 0xb0, 0x0, 0x0, 0xa4, 0xea, 0x12, 0x0, 0x8,
			0x71, 0xd0, 0x1, 0x78, 0x75, 0x15, 0x1, 0x84, 0xe9, 0x12, 0x0, 0xa, 0x0, 0x0, 0x0, 0x6d,
			0xee, 0x12, 0x0, 0x97, 0xe9, 0x12, 0x0, 0xd8, 0xe9, 0x12, 0x0, 0x18, 0xfa, 0x52, 0x0, 0xa,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x24, 0x0, 0x0, 0x0, 0xe5, 0x98, 0x53, 0x0, 0x64,
			0xd, 0x4f, 0x0, 0x3, 0x0, 0x1, 0x77, 0x0, 0x0, 0x0, 0x0, 0x58, 0xd3, 0x17, 0x0, 0x1,
			0x0, 0x0, 0x0, 0x98, 0xe9, 0x12, 0x0, 0xe0, 0xe9, 0x12, 0x0, 0x24, 0x0, 0x0, 0x0, 0x4c,
			0xef, 0x12, 0x0, 0xd0, 0xe9, 0x12, 0x0, 0x48, 0xbf, 0x52, 0x0, 0x5c, 0xef, 0x12, 0x0, 0x4,
			0xea, 0x12, 0x0, 0x9, 0x0, 0x0, 0x0, 0xe6, 0x98, 0x53, 0x0, 0x4, 0xea, 0x12, 0x0, 0xec,
			0x0, 0x12, 0x0, 0xd7, 0xc2, 0x52, 0x0, 0x4, 0xea, 0x12, 0x0, 0x9, 0x0, 0x0, 0x0, 0x4c,
			0x3e, 0xae, 0x0, 0x0}}
	for _, expected := range expecteds {
		var s cmdSpreadCond
		err := s.Unmarshal(expected)
		if err != nil {
			t.Error(err)
			return
		}
		actual, err := s.Marshal()
		if err != nil {
			t.Error(err)
			return
		}
		if len(actual) != len(expected) {
			t.Errorf("len mismatch expected %d vs %d %#v actual %#v", len(expected), len(actual), expected, actual)
			return
		}
		for i := 0; i < len(actual); i++ {
			if actual[i] != expected[i] {
				t.Errorf("byte mismatch expected %#v actual %#v", expected, actual)
				return
			}
		}
	}
}
