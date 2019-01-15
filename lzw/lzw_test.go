package lzw

import (
    "strings"
    "bytes"
    "testing"
)

func TestSingleString(t *testing.T) {
    tests := []string {
        "Hello World! I really like to say Hello to this World!",
        "ABABABA", // test special case when encoder is ahead of decoder
    }

    for _, test := range tests {
        originalData := test
        r := strings.NewReader(originalData)

        encodedData, nbits := Compress(r)

        r2 := bytes.NewReader(*encodedData)
        decodedData := Uncompress(r2, nbits)

        if len(originalData) != len(*decodedData) {
            t.Errorf("Compression failed, mismatch on len(): %d vs. %d",
                len(originalData), len(*decodedData))
        }

        for i := 0 ; i < len(originalData) ; i++ {
            if originalData[i] != (*decodedData)[i] {
                t.Errorf("Compression failed")
            }
        }
    }
}

func TestSpecialCase(t *testing.T) {
}
