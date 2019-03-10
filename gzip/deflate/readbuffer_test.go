package deflate

import (
    "testing"
    "strings"
    "bytes"
    "math/rand"
    "strconv"
    "fmt"
)

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

func TestRewind(t *testing.T) {
    shakespeare := "To thine own self be true, and it must follow, as the night the day, thou canst not then be false to any man."
    r := strings.NewReader(shakespeare)
    bufferSize := 16
    rb := NewReadBuffer(r, bufferSize)

    // Load
    if err := rb.LoadMoreBytes(); err != nil {
        t.Error(err.Error())
    }

    // Those numbers are totally arbitrary
    var readCount, rewindCount int = 85, 47

    for i := 0; i < readCount; i++ {
        rb.ReadBit()
    }

    if err := rb.Rewind(uint(rewindCount)); err != nil {
        t.Error(err.Error())
    }

    if rb.index != int((readCount-rewindCount)/8) || rb.bitPosition != int((readCount-rewindCount)%8) {
        t.Errorf("Rewinding the buffer didn't work as expected.")
    }

    // Testing rewinding to zero
    if err := rb.Rewind(uint(readCount-rewindCount)); err != nil {
        t.Error(err.Error())
    }

    if rb.index != 0 || rb.bitPosition != 0 {
        t.Errorf("Rewinding the buffer didn't work as expected.")
    }

    // Test creating a rewinding error
    if err := rb.Rewind(uint(1)); err == nil {
        t.Errorf("Rewinding beyond the start of the buffer should have failed.")
    }
}

func TestBytesToUint64(t *testing.T) {
    rand.Seed(3)
    for numBytes := 1; numBytes <= 9; numBytes++ {
        array := make([]byte, 9)
        var buffer bytes.Buffer
        for i := 0; i < numBytes; i++ {
            array[i] = byte(rand.Int())
            buffer.WriteString(fmt.Sprintf("%0*s", 8, strconv.FormatUint(uint64(array[i]), 2)))
        }
        for i := numBytes; i < 9; i++ {
            buffer.WriteString("00000000")
        }
        originalString := buffer.String()

        for i := 0; i <= 8; i++ {
            converted := BytesToUint64(array, i)
            targetString := fmt.Sprintf("%0*s", 64, strconv.FormatUint(converted, 2))
            if bytes.Equal([]byte(originalString[i:64+i]), []byte(targetString)) == false {
                t.Errorf("Failed to convert byte slice to uint64:\n%s\n%s\n", originalString[i:64+i], targetString)
            }
        }
    }
}
