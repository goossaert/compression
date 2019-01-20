package lzw

import (
    "io"
    "strings"
    "encoding/binary"

    "github.com/goossaert/compression/logging"
)

// TODO profile memory allocations
// TODO code duplication in the flush part of Compress()
// TODO use varints instead of uint16

func Compress(rawData io.Reader) (compressedData *[]byte, nbits int) {
    var out []byte
    logging.Trace.Printf("Compress()\n")
    stringsToCodes := make(map[string]uint16)
    for i := 0 ; i < 256 ; i++ {
        stringsToCodes[string(i)] = uint16(i)
    }
    var nextCode uint16 = 256

    readBuffer := make([]byte, 1024)
    outputBytes := make([]byte, 2)
    nbits = 0
    var window []byte
    for {
        n, err := rawData.Read(readBuffer)
        if err == io.EOF {
            // Flush out the last string to encode
            outputCode := stringsToCodes[string(window[:])]
            logging.Trace.Printf("ENC %s => %d\n", string(window[:]), outputCode)
            binary.LittleEndian.PutUint16(outputBytes, outputCode)
            out = append(out, outputBytes...)
            nbits += 16
            break
        }
        logging.Trace.Printf("nbytes read: %d\n", n)
        for i := 0 ; i < n ; i++ {
            window = append(window, readBuffer[i])
            if _, ok := stringsToCodes[string(window[:])]; ok {
                continue
            }

            stringsToCodes[string(window[:])] = nextCode
            logging.Trace.Printf("ADD %s => %d\n", string(window[:]), nextCode)
            nextCode += 1
            outputCode := stringsToCodes[string(window[:len(window)-1])]
            logging.Trace.Printf("ENC %s => %d\n", string(window[:len(window)-1]), outputCode)
            binary.LittleEndian.PutUint16(outputBytes, outputCode)
            out = append(out, outputBytes...)
            nbits += 16

            window = window[:1]
            window[0] = readBuffer[i]
        }
    }

    return &out, nbits
}


func Uncompress(compressedData io.Reader, nbits int) (uncompressedData *[]byte) {
    var out []byte
    logging.Trace.Printf("Uncompress()\n")
    codesToStrings := make(map[uint16]string)
    for i := 0 ; i < 256 ; i++ {
        codesToStrings[uint16(i)] = string(i)
    }
    var nextCode uint16 = 256

    var stringBuilder strings.Builder
    readBuffer := make([]byte, 1024)
    var previousString string

    for {
        n, err := compressedData.Read(readBuffer)
        if err == io.EOF {
            break
        }
        for i := 0 ; i < n ; i += 2 {
            var code uint16 = binary.LittleEndian.Uint16(readBuffer[i:i+2])
            if _, ok := codesToStrings[code] ; !ok {
                stringBuilder.Reset()
                stringBuilder.WriteString(previousString)
                stringBuilder.WriteByte(previousString[0])
                codesToStrings[code] = stringBuilder.String()
                logging.Trace.Printf("ADDN %s => %d\n", stringBuilder.String(), code)
            }
            out = append(out, codesToStrings[code]...)
            logging.Trace.Printf("DEC %d => %s\n", code, codesToStrings[code])
            if len(previousString) > 0 {
                stringBuilder.Reset()
                stringBuilder.WriteString(previousString)
                stringBuilder.WriteByte(codesToStrings[code][0])
                codesToStrings[nextCode] = stringBuilder.String()
                logging.Trace.Printf("ADD %s => %d\n", stringBuilder.String(), nextCode)
                nextCode += 1
            }
            previousString = codesToStrings[code]
        }
    }
    return &out
}

