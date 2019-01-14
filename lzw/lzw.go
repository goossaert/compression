package lzw

import (
    "fmt"
    "io"
    "strings"
    "encoding/binary"
)

// TODO need more idiomatic use of byte slides and strings
// TODO profile memory allocations
// TODO code duplication in the flush part of Compress()
// TODO use varints instead of uint16
// TODO check if really need sizeWindow or could use len(window)

func Compress(rawData io.Reader) (compressedData *[]byte, nbits int) {
    out := make([]byte, 0)
    fmt.Printf("Compress()")
    stringsToCodes := make(map[string]uint16)
    for i := 0 ; i < 256 ; i++ {
        stringsToCodes[string(i)] = uint16(i)
    }
    var nextCode uint16 = 256

    readBuffer := make([]byte, 1024)
    outputBytes := make([]byte, 2)
    nbits = 0
    window := make([]byte, 0)
    sizeWindow := 0
    for {
        n, err := rawData.Read(readBuffer)
        if err == io.EOF {
            // Flush out the last string to encode
            outputCode := stringsToCodes[string(window[:sizeWindow-1])]
            binary.LittleEndian.PutUint16(outputBytes, uint16(outputCode))
            out = append(out, outputBytes[0], outputBytes[1])
            nbits += 16
            break
        }
        for i := 0 ; i < n ; i++ {
            window = append(window, readBuffer[i])
            sizeWindow += 1
            if _, exists := stringsToCodes[string(window[:sizeWindow])]; exists == true {
                continue
            }

            stringsToCodes[string(window[:sizeWindow])] = nextCode
            nextCode += 1
            outputCode := stringsToCodes[string(window[:sizeWindow-1])]
            binary.LittleEndian.PutUint16(outputBytes, uint16(outputCode))
            out = append(out, outputBytes[0], outputBytes[1])
            nbits += 16

            sizeWindow = 1
            window = window[:1]
            window[0] = readBuffer[i]
        }
    }

    return &out, nbits
}


func Uncompress(compressedData io.Reader, nbits int) (uncompressedData *[]byte) {
    out := make([]byte, 0)
    fmt.Printf("Uncompress()")
    codesToStrings := make(map[uint16]string)
    for i := 0 ; i < 256 ; i++ {
        codesToStrings[uint16(i)] = string(i)
    }
    var nextCode uint16 = 256

    var stringBuilder strings.Builder
    readBuffer := make([]byte, 1024)
    var previousString *string

    for {
        n, err := compressedData.Read(readBuffer)
        if err == io.EOF {
            break
        }
        for i := 0 ; i < n ; i += 2 {
            var code uint16 = binary.LittleEndian.Uint16(readBuffer[i:i+2])
            if _, exists := codesToStrings[code] ; exists == false {
                stringBuilder.Reset()
                stringBuilder.WriteString(*previousString)
                stringBuilder.WriteByte((*previousString)[0])
                codesToStrings[code] = stringBuilder.String()
            }
            out = append(out, codesToStrings[code]...)
            if previousString != nil && len(*previousString) > 0 {
                stringBuilder.Reset()
                stringBuilder.WriteString(*previousString)
                stringBuilder.WriteByte(codesToStrings[code][0])
                codesToStrings[nextCode] = stringBuilder.String()
                nextCode += 1
            }
            newString := codesToStrings[code]
            previousString = &newString
        }
    }
    return &out
}

