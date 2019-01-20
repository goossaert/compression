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

        if bytes.Equal([]byte(originalData), *decodedData) == false {
            t.Errorf("Compression failed")
        }
    }
}

func BenchmarkString(b *testing.B) {
    originalData := "Hello World! I really like to say Hello to this World!"

    for i := 0 ; i < b.N ; i++ {
        r := strings.NewReader(originalData)
        encodedData, nbits := Compress(r)

        r2 := bytes.NewReader(*encodedData)
        Uncompress(r2, nbits)
    }
}
