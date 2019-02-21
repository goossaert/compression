package deflate

import (
    "testing"
    "strings"
    "bytes"
    //"fmt"
)

func MaxInt(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func TestLoadingBytes(t *testing.T) {
    shakespeare := "To thine own self be true, and it must follow, as the night the day, thou canst not then be false to any man."
    r := strings.NewReader(shakespeare)
    bufferSize := 16
    rb := NewReadBuffer(r, bufferSize)

    for i := 0; i < len(shakespeare); i += bufferSize {

        // Load
        if err := rb.LoadMoreBytes(); err != nil {
            t.Error(err.Error())
        }

        // Compare
        endIndexShake := i + bufferSize
        if endIndexShake > len(shakespeare) {
            endIndexShake = i + len(shakespeare) % bufferSize
        }
        endIndexBuf := endIndexShake % bufferSize
        if endIndexBuf == 0 {
            endIndexBuf = bufferSize
        }
        //fmt.Printf("[%s] and [%s]\n", shakespeare[i:endIndexShake], string(rb.buf[0:endIndexBuf]))
        if bytes.Equal([]byte(shakespeare[i:endIndexShake]), rb.buf[0:endIndexBuf]) == false {
            t.Errorf("Data loading failed.")
            return
        }

        // Consume
        for i := 0; i < endIndexBuf*8; i++ {
            rb.ReadBit()
        }
    }
}
