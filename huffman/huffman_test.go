package huffman

import (
    "strings"
    "bytes"
    "testing"
)

func TestSingleString(t *testing.T) {
    originalData := "Hello World!"

    r := strings.NewReader(originalData)
    htree := BuildHTree(r)
    htree.Print()

    r2 := strings.NewReader(originalData)
    encodedData, nbits := htree.EncodeBytes(r2)

    /*
    import "strconv"
    extraByte := 0
    if nbits % 8 > 0 {
        extraByte = 1
    }
    for i := 0 ; i < int(nbits/8) + extraByte ; i++ {
        fmt.Printf("%0*s|", 8, strconv.FormatUint(uint64((*encodedData)[i]), 2))
    }
    */

    decodedData := htree.DecodeBytes(*encodedData, nbits)

    if bytes.Equal([]byte(originalData), *decodedData) == false {
        t.Errorf("Compression failed")
    }
}


func BenchmarkSingleString(b *testing.B) {
    originalData := "Hello World!"

    for i := 0 ; i < b.N ; i++ {
        r := strings.NewReader(originalData)
        htree := BuildHTree(r)
        htree.Print()

        r2 := strings.NewReader(originalData)
        encodedData, nbits := htree.EncodeBytes(r2)
        htree.DecodeBytes(*encodedData, nbits)
    }

}


func TestAllAlphabet(t *testing.T) {
    originalData := make([]byte, 256, 256)
    for i := 0 ; i < 256 ; i++ {
        originalData[i] = byte(i)
    }

    r := bytes.NewReader(originalData)
    htree := BuildHTree(r)
    htree.Print()

    r2 := bytes.NewReader(originalData)
    encodedData, nbits := htree.EncodeBytes(r2)

    decodedData := htree.DecodeBytes(*encodedData, nbits)

    if bytes.Equal([]byte(originalData), *decodedData) == false {
        t.Errorf("Compression failed")
    }
}
