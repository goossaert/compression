package deflate

import (
    "io"
    //"fmt"
)

type translationItem struct {
    code int
    extraBit int
    minRange int
    maxRange int
}

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


func GenerateCanonicalPrefixes(codeLengths []int) ([]uint32) {
    // Port of Peter Deutsch's original C function from RFC1951

    maxCodeLength := 0
    for _, codeLength := range codeLengths {
        if codeLength > maxCodeLength {
            maxCodeLength = codeLength
        }
    }

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

    codes := make([]uint32, len(codeLengths))
    for i, codeLength := range codeLengths {
        if codeLength > 0 {
            codes[i] = uint32(nextCode[codeLength]) << uint(32-codeLength)
            nextCode[codeLength] += 1
        }
    }

    return codes
}


type Translator struct {
    litLenDecodingTables map[int](map[uint32]translationItem)
    litLenMinBits int
    litLenMaxBits int
    distanceDecodingTables map[int](map[uint32]translationItem)
    distanceMinBits int
    distanceMaxBits int
}

func NewTranslator(litLenSeq []int, distanceSeq []int) *Translator {
    t := new(Translator)

    // Generates hash tables to translate prefixes to literals/lengths
    t.litLenDecodingTables = make(map[int](map[uint32]translationItem))
    litLenCodes := GenerateCanonicalPrefixes(litLenSeq)

    for i := 0; i <= 256; i++ {
        numBits := litLenSeq[i]
        if _, ok := t.litLenDecodingTables[numBits]; !ok {
            t.litLenDecodingTables[numBits] = make(map[uint32]translationItem)
        }
        t.litLenDecodingTables[numBits][litLenCodes[i]] = translationItem{i, 0, 0, 0}
    }

    for i := 0; i < len(latLenTable); i++ {
        numBits := litLenSeq[i+257]
        if _, ok := t.litLenDecodingTables[numBits]; !ok {
            t.litLenDecodingTables[numBits] = make(map[uint32]translationItem)
        }
        t.litLenDecodingTables[numBits][litLenCodes[i+257]] = latLenTable[i]
    }

    // Generates hash table to translate prefixes to distances
    t.distanceDecodingTables = make(map[int](map[uint32]translationItem))
    t.distanceDecodingTables[5] = make(map[uint32]translationItem)
    distanceCodes := GenerateCanonicalPrefixes(distanceSeq)
    for i := 0; i < len(distanceTable); i++ {
        t.distanceDecodingTables[5][distanceCodes[i]] = distanceTable[i]
    }

    return t
}


func (t *Translator) decodePrefix(prefix *Prefix) (numBitsRead, matchType, litLen, distance int) {
    return 0, 1, 0, 0
}


func decodeStream(reader io.Reader, writer io.Writer) {

    litLenSequence := GenerateMode2LitLenSequence()
    distanceSequence := GenerateMode2DistanceSequence()
    NewTranslator(litLenSequence, distanceSequence)

    /*
    rb := NewReadBuffer(reader, 4096)
    isValid := true
    for isValid {
        if rb.BitsLeftToRead() < 32 {
            if err := rb.LoadMoreBytes(); err != nil {
                //
            }
        }

        rb.ReadBit()
    }
    */
}

