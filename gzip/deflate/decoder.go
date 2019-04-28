package deflate

import (
    "io"
    "math/bits"
    "fmt"
    "strconv"
    "encoding/binary"
    "errors"
)

type translationItem struct {
    code int
    numExtraBits int
    minRange int
    maxRange int
}

const (
    DeflateNoCompression = 0
    DeflateFixed = 1
    DeflateDynamic = 2
    DeflateReserved = 3
)

var latLenTable = []translationItem{
                    {257, 0,   3,   3},
                    {258, 0,   4,   4},
                    {259, 0,   5,   5},
                    {260, 0,   6,   6},
                    {261, 0,   7,   7},
                    {262, 0,   8,   8},
                    {263, 0,   9,   9},
                    {264, 0,  10,  10},
                    {265, 1,  11,  12},
                    {266, 1,  13,  14},
                    {267, 1,  15,  16},
                    {268, 1,  17,  18},
                    {269, 2,  19,  22},
                    {270, 2,  23,  26},
                    {271, 2,  27,  30},
                    {272, 2,  31,  34},
                    {273, 3,  35,  42},
                    {274, 3,  43,  50},
                    {275, 3,  51,  58},
                    {276, 3,  59,  68},
                    {277, 4,  67,  82},
                    {278, 4,  83,  98},
                    {279, 4,  99, 114},
                    {280, 4, 115, 130},
                    {281, 5, 131, 162},
                    {282, 5, 163, 194},
                    {283, 5, 195, 226},
                    {284, 5, 227, 257},
                    {285, 0, 258, 258},
                }

var distanceTable = []translationItem{
                    { 0,  0,     1,     1},
                    { 1,  0,     2,     2},
                    { 2,  0,     3,     3},
                    { 3,  0,     4,     4},
                    { 4,  1,     5,     6},
                    { 5,  1,     7,     8},
                    { 6,  2,     9,    12},
                    { 7,  2,    13,    16},
                    { 8,  3,    17,    24},
                    { 9,  3,    25,    32},
                    {10,  4,    33,    48},
                    {11,  4,    49,    64},
                    {12,  5,    65,    96},
                    {13,  5,    97,   128},
                    {14,  6,   129,   192},
                    {15,  6,   193,   256},
                    {16,  7,   257,   384},
                    {17,  7,   385,   512},
                    {18,  8,   513,   768},
                    {19,  8,   769,  1024},
                    {20,  9,  1025,  1536},
                    {21,  9,  1537,  2048},
                    {22, 10,  2049,  3072},
                    {23, 10,  3073,  4096},
                    {24, 11,  4097,  6184},
                    {25, 11,  6185,  8192},
                    {26, 12,  8193, 12288},
                    {27, 12, 12289, 16384},
                    {28, 13, 16385, 24576},
                    {29, 13, 24577, 32768},
                }

func generateUint64BitMasks() ([]uint64, []uint64) {
    leftBitMasks := make([]uint64, 65)
    rightBitMasks := make([]uint64, 65)

    leftBitMasks[0] = 0
    rightBitMasks[0] = 0

    for i := 1; i <= 64; i++ {
        leftBitMasks[i] = uint64(0xffffffffffffffff) << uint(64-i)
        rightBitMasks[i] = uint64(0xffffffffffffffff) >> uint(64-i)
        //fmt.Printf("generateUint64BitMasks() Left  %d %0*s\n", i, 64, strconv.FormatUint(leftBitMasks[i], 2))
        //fmt.Printf("generateUint64BitMasks() Right %d %0*s\n", i, 64, strconv.FormatUint(rightBitMasks[i], 2))
    }

    return leftBitMasks, rightBitMasks
}


func GenerateMode2LitLenSequence() []int {
    seq := make([]int, 288)
    for i := 0; i < 144; i++ {
        seq[i] = 8
    }

    for i := 144; i < 144+112; i++ {
        seq[i] = 9
    }

    for i := 144+112; i < 144+112+24; i++ {
        seq[i] = 7
    }

    for i := 144+112+24; i < 144+112+24+8; i++ {
        seq[i] = 8
    }

    return seq
}


func GenerateMode2DistanceSequence() []int {
    seq := make([]int, 30)
    for i := 0; i < 30; i++ {
        seq[i] = 5
    }
    return seq
}


func GetMinMaxSlice(s []int) (int, int) {
    if len(s) == 0 {
        return 0, 0
    }

    min, max := s[0], s[0]
    for i := 1; i < len(s); i++ {
        if s[i] > max {
            max = s[i]
        }
        if s[i] < min {
            min = s[i]
        }
    }

    return min, max
}


func GenerateCanonicalPrefixes(codeLengths []int) ([]uint64) {
    // Port of Peter Deutsch's original C function from RFC1951

    _, maxCodeLength := GetMinMaxSlice(codeLengths)

    blCount := make([]int, maxCodeLength+1)
    for _, codeLength := range codeLengths {
        blCount[codeLength] += 1
    }

    code := 0
    nextCode := make([]int, maxCodeLength+1)
    blCount[0] = 0
    for bits := 1; bits <= maxCodeLength; bits++ {
        code = (code + blCount[bits-1]) << 1;
        nextCode[bits] = code
    }

    codes := make([]uint64, len(codeLengths))
    for i, codeLength := range codeLengths {
        if codeLength > 0 {
            codes[i] = uint64(nextCode[codeLength]) << uint(64-codeLength)
            nextCode[codeLength] += 1
        }
    }

    return codes
}


type Translator struct {
    litLenDecodingTables map[int](map[uint64]translationItem)
    litLenMinBits int
    litLenMaxBits int
    distanceDecodingTables map[int](map[uint64]translationItem)
    distanceMinBits int
    distanceMaxBits int
    leftBitMasks []uint64
    rightBitMasks []uint64
}

func NewTranslator(litLenSeq []int, distanceSeq []int) *Translator {
    t := new(Translator)

    // Generates hash tables to translate prefixes to literals/lengths
    t.litLenDecodingTables = make(map[int](map[uint64]translationItem))
    litLenCodes := GenerateCanonicalPrefixes(litLenSeq)
    t.litLenMinBits, t.litLenMaxBits = GetMinMaxSlice(litLenSeq)

    for i := t.litLenMinBits; i <= t.litLenMaxBits; i++ {
        t.litLenDecodingTables[i] = make(map[uint64]translationItem)
    }

    for i := 0; i <= 256; i++ {
        numBits := litLenSeq[i]
        //if _, ok := t.litLenDecodingTables[numBits]; !ok {
        //    t.litLenDecodingTables[numBits] = make(map[uint64]translationItem)
        //}
        t.litLenDecodingTables[numBits][litLenCodes[i]] = translationItem{i, 0, 0, 0}
    }

    for i := 0; i < len(latLenTable); i++ {
        numBits := litLenSeq[i+257]
        //if _, ok := t.litLenDecodingTables[numBits]; !ok {
        //    t.litLenDecodingTables[numBits] = make(map[uint64]translationItem)
        //}
        t.litLenDecodingTables[numBits][litLenCodes[i+257]] = latLenTable[i]
    }

    // Generates hash table to translate prefixes to distances
    // TODO right now it's only a size of 5 bits because I'm testing the fixed encoding mode only,
    //      but this will have to be adjusted for dynamic mode.
    t.distanceDecodingTables = make(map[int](map[uint64]translationItem))
    t.distanceDecodingTables[5] = make(map[uint64]translationItem)
    distanceCodes := GenerateCanonicalPrefixes(distanceSeq)
    for i := 0; i < len(distanceTable); i++ {
        t.distanceDecodingTables[5][distanceCodes[i]] = distanceTable[i]
    }
    t.distanceMinBits = 5
    t.distanceMaxBits = 5

    // Generate bit masks
    t.leftBitMasks, t.rightBitMasks = generateUint64BitMasks()

    return t
}


func (t *Translator) decodePrefix(prefix uint64) (numBitsRead uint, isLiteral bool, litLen, distance int, err error) {
    numBitsRead = 0
    litLen = 0
    distance = 0
    litLenFound := false
    isLiteral = false
    for numBits := t.litLenMinBits; numBits <= t.litLenMaxBits; numBits++ {
        maskedPrefix := prefix & t.leftBitMasks[numBits]
        if item, ok := t.litLenDecodingTables[numBits][maskedPrefix]; ok {
            fmt.Printf("LitLen %0*s - %d\n", 64, strconv.FormatUint(maskedPrefix, 2), item.code)
            if item.code <= 256 {
                litLen = item.code
                isLiteral = true
                fmt.Printf("  => Literal '%s'\n", string(litLen))
            } else {
                extraBits := int(bits.Reverse64(prefix << uint(numBits)) & t.rightBitMasks[item.numExtraBits])
                litLen = item.minRange + extraBits
                fmt.Printf("  => %0*s\n", 64, strconv.FormatUint(uint64(extraBits), 2))
                fmt.Printf("  => Length %d\n", litLen)
            }
            numBitsRead += uint(numBits + item.numExtraBits)
            litLenFound = true
            break
        }
    }

    if litLenFound == false {
        // Invalid input data
    }

    if isLiteral {
        return numBitsRead, isLiteral, litLen, 0, nil
    }

    prefix = prefix << numBitsRead
    distanceFound := false
    for numBits := t.distanceMinBits; numBits <= t.distanceMaxBits; numBits++ {
        maskedPrefix := prefix & t.leftBitMasks[numBits]
        if item, ok := t.distanceDecodingTables[numBits][maskedPrefix]; ok {
            extraBits := int((bits.Reverse64(prefix) >> uint(numBits)) & t.rightBitMasks[item.numExtraBits])
            distance = item.minRange + extraBits
            numBitsRead += uint(numBits + item.numExtraBits)
            fmt.Printf("Distance %0*s\n", 64, strconv.FormatUint(maskedPrefix, 2))
            fmt.Printf("  => %0*s\n", 64, strconv.FormatUint(uint64(extraBits), 2))
            fmt.Printf("  => %d\n", distance)
            distanceFound = true
            break
        }
    }

    if distanceFound == false {
        // Invalid input data
    }

    return numBitsRead, isLiteral, litLen, distance, nil
}


func copyBytes(wb *WriteBuffer, rb *ReadBuffer, n int) error {
    i := 0
    for i < n {
        step := 1024
        if i + step > n {
            step = n - i
        }
        data, numBytesRead, err := rb.ReadAlignedBytes(step)
        if err != nil {
            return err
        }

        wb.WriteBytes(data)
        i += numBytesRead
    }
    return nil
}


//func DecodeStream(reader io.Reader, writer io.Writer) error {
func DecodeStream(rb *ReadBuffer, writer io.Writer) error {

    fmt.Printf("DecodeStream()\n");

    //rb := NewReadBuffer(reader, 4096)
    wb := NewWriteBuffer(writer, 32768)

    if rb.BitsLeftToRead() < 64 {
        if err := rb.LoadMoreBytes(); err != nil {
            return err
        }
    }

    var prefix uint64
    var err error

    isLastBlock := false

    for !isLastBlock {
        // Read Deflate block header
        if prefix, err = rb.Peek(); err != nil {
            return err
        }
        if int(prefix >> 63) == 1 {
            isLastBlock = true
        }
        // According to RFC 1951, "Data elements other than Huffman codes are
        // packed starting with the least-significant bit of the data element".
        // Because Peek() reverses the bits in the bytes, here the two bits of
        // the compression mode need to be reversed again.
        // The line below turns [b63 b62 b61 ... b2 b1 b0] into [0 0 0 ... b61 b62]
        var compressionMode int = int(prefix >> 62) & 0x1 | int(prefix >> 60) & 0x2
        rb.Forward(3)

        fmt.Printf("Last block: %v, compression mode: %d\n", isLastBlock, compressionMode)

        // Decodes data based on compression mode
        if compressionMode == DeflateNoCompression {
            // From RFC 1951, 3.2.4. "Any bits of input up to
            // the next byte boundary are ignored."
            rb.Forward(uint(8 - rb.bitPosition))

            uncompressedHeader, _, err := rb.ReadAlignedBytes(4)
            if err != nil {
                return err
            }

            length := binary.LittleEndian.Uint16(uncompressedHeader[0:2])
            lengthOneComplement := binary.LittleEndian.Uint16(uncompressedHeader[2:4])
            if length != ^lengthOneComplement {
                return errors.New("Found invalid length of block\n")
            }

            if err := copyBytes(wb, rb, int(length)); err != nil {
                return err
            }
        } else if compressionMode == DeflateReserved {
            //
        } else {
            var translator *Translator
            if compressionMode == DeflateFixed {
                litLenSequence := GenerateMode2LitLenSequence()
                distanceSequence := GenerateMode2DistanceSequence()
                translator = NewTranslator(litLenSequence, distanceSequence)
            } else if compressionMode == DeflateDynamic {
                // 
            }

            hasMoreData := true
            for hasMoreData {
                if rb.BitsLeftToRead() < 64 {
                    if err = rb.LoadMoreBytes(); err != nil {
                        return err
                    }
                }

                if prefix, err = rb.Peek(); err != nil {
                    return err
                }

                numBitsRead, isLiteral, litLen, distance, err := translator.decodePrefix(prefix)
                if err != nil {
                    return err
                }

                if isLiteral {
                    if litLen == 256 {
                        hasMoreData = false
                    } else {
                        wb.WriteByte(byte(litLen))
                    }
                } else {
                    wb.RepeatBytes(litLen, distance)
                }

                rb.Forward(numBitsRead)
            }
        }
    }

    wb.Flush()
    return nil
}

