package main

import (
    "fmt"
    "os"
    "io"
    "log"
    "encoding/binary"
    "time"
    "hash/crc32"
    "compress/gzip"
    "errors"
    "github.com/goossaert/compression/gzip/deflate"
)

// Gzip constants
const (
    FlagAscii = 1
    HeaderCrc16Present = 2
    ExtraFieldPresent = 4
    OriginalFileNamePresent = 8
    FileCommentPresent = 16

    GzipMagic1 = 0x1f
    GzipMagic2 = 0x8b
)

// Deflate constants
const (
    MaxBlockSizeNoCompress = 65535
)


type GzipHeader struct {
    // Mandatory header fields
    magicHeader uint16
    compressionMethod uint8
    flags uint8
    fileLastModifiedTimestamp uint32
    extraFlags uint8
    operatingSystem uint8

    // Optional header fields
    partNumber uint16
    extraField []byte
    originalFilename string
    fileComment string
    headerCRC16 uint16

    // Compressed data
    compressedData []byte

    // End of file
    checksum uint32 // crc32
    uncompressedInputSize uint32
}


func WriteGzipNoCompression(w io.Writer, data []byte) (err error) {
    if len(data) > MaxBlockSizeNoCompress {
        return errors.New(fmt.Sprintf("Doesn't support streams above %d bytes\n", MaxBlockSizeNoCompress))
    }

    gzipHeader := make([]byte, 10)
    gzipHeader[0] = GzipMagic1
    gzipHeader[1] = GzipMagic2
    gzipHeader[2] = 8 // deflate
    gzipHeader[3] = 0 // flags
    binary.LittleEndian.PutUint32(gzipHeader[4:8], uint32(time.Now().Unix()))
    gzipHeader[8] = 0 // leave extra flags empty
    gzipHeader[9] = 255 // Operating System - 255 means Unknown

    // Deflate mode 0 header
    deflateHeader := make([]byte, 5)
    deflateHeader[0] = 0x01 // first three bits are 100, stored with least-significant bits first
    binary.LittleEndian.PutUint16(deflateHeader[1:3], uint16(len(data)))
    binary.LittleEndian.PutUint16(deflateHeader[3:5], uint16(^len(data)))

    var checksum uint32 = 0
    checksum = crc32.Update(checksum, crc32.IEEETable, data)
    gzipFooter := make([]byte, 8)
    binary.LittleEndian.PutUint32(gzipFooter[0:4], checksum)
    binary.LittleEndian.PutUint32(gzipFooter[4:8], uint32(len(data)))

    if _, err = w.Write(gzipHeader); err != nil {
        return err
    }

    if _, err = w.Write(deflateHeader); err != nil {
        return err
    }

    if _, err = w.Write(data); err != nil {
        return err
    }

    if _, err = w.Write(gzipFooter); err != nil {
        return err
    }

    return nil
}


func GzipReader(filepath string) error {
    file, err := os.Open(filepath)
    rb := deflate.NewReadBuffer(file, 4096)

    if err = rb.LoadMoreBytes(); err != nil {
        return err
    }

    header, numBytesRead, err := rb.ReadAlignedBytes(10)
    if err != nil {
        return err
    }
    if numBytesRead != 10 {
        errors.New("Invalid file")
    }

    gzipNumber1 := header[0]
    gzipNumber2 := header[1]
    compressionMethod := header[2]
    flagText        := header[3] & 0x01
    flagHeaderCRC   := header[3] & 0x02
    flagExtraFields := header[3] & 0x04
    flagName        := header[3] & 0x08
    flagComment     := header[3] & 0x10

    modificationTime := binary.LittleEndian.Uint32(header[4:8])

    extraFlags := header[8]
    operatingSystem := header[9]

    if gzipNumber1 != GzipMagic1 || gzipNumber2 != GzipMagic2 {
        errors.New("Invalid file")
    }

    fmt.Printf("Gzip magic numbers 0x%x 0x%x\n", gzipNumber1, gzipNumber2)

    if compressionMethod != 8 {
        errors.New("Unknown compression method, expected 'deflate'")
    }

    if flagExtraFields == 1 {
        temp, _, err := rb.ReadAlignedBytes(2)
        if err != nil {
            return err
        }
        lenExtra := binary.LittleEndian.Uint16(temp)
        if _, _, err := rb.ReadAlignedBytes(int(lenExtra)); err != nil {
            return err
        }
    }

    stringReader := func(rb *deflate.ReadBuffer) (error, string) {
        var temp []byte
        for true {
            if b, err := rb.ReadAlignedByte(); err != nil {
                return err, string("")
            } else {
                if b == 0 {
                    break
                }
                temp = append(temp, b)
            }
        }
        return nil, string(temp)
    }

    // Read filename and comment if present
    var filename, comment string
    if flagName == 1 {
        if err, filename = stringReader(rb); err != nil {
            return err
        }
    }
    if flagComment == 1 {
        if err, comment = stringReader(rb); err != nil {
            return err
        }
    }

    if flagHeaderCRC == 1 {
        // Ignoring the Header CRC
        _, _, err := rb.ReadAlignedBytes(2)
        if err != nil {
            return err
        }
    }

    // At this stage, the read buffer 'rb' is at the correct
    // reading index to access the compressed data

    outfile, err := os.Create("./decompressed-data")
    defer outfile.Close()

    if err := deflate.DecodeStream(rb, outfile); err != nil {
        return err
    }

    fmt.Println("Modification time", time.Unix(int64(modificationTime), 0).Format(time.RFC822Z))
    fmt.Printf("Filename: %s, Comment: %s\n", filename, comment)

    if flagText == 1 && flagHeaderCRC == 1 && flagExtraFields == 1 && flagName == 1 && flagComment == 1 {
        //
    }

    fmt.Printf("Flags: text:%d, header:%d, extraFields:%d, name:%d, comment:%d\n", flagText, flagHeaderCRC, flagExtraFields, flagName, flagComment)

    if extraFlags == 1 && operatingSystem == 1 {
        //
    }

    //b, _ := rb.ReadAlignedByte()
    //fmt.Printf("byte %d\n", b)

    fmt.Printf("out\n")
    return nil
}


func main() {
    data := "aaaaabcdefghijbbbbbbbbbbbbbbbbbbbbbaaaaabbb"
    filepath := "./myfile-custom.gz"
    file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
    if err != nil {
       log.Fatal(err)
    }
    defer file.Close()

    err = WriteGzipNoCompression(file, []byte(data))
    if err != nil {
        log.Fatal(err)
    }

    f, _ := os.Create("./myfile-stdlib.gz")
    defer f.Close()
    w, _ := gzip.NewWriterLevel(f, gzip.NoCompression)
    //w := gzip.NewWriter(f)
    w.Write([]byte(data))
    w.Close()

    if err := GzipReader("./myfile-stdlib.gz"); err != nil {
        log.Fatal(err)
    }
}
