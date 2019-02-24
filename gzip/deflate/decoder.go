package deflate

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

