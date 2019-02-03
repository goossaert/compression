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
        return errors.New(fmt.Sprintf("Doest not support streams above %d bytes\n", MaxBlockSizeNoCompress))
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
    deflateHeader := make([]byte, 1)
    deflateHeader[0] = 1 // first three bits are 100, stored with least-significant bits first
    deflateSizes := make([]byte, 4)
    binary.LittleEndian.PutUint16(deflateSizes[0:2], uint16(len(data)))
    binary.LittleEndian.PutUint16(deflateSizes[2:4], uint16(^len(data)))

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

    if _, err = w.Write(deflateSizes); err != nil {
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


func main() {
    data := "aaaaaaaaaa"
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
    defer w.Close()
    w.Write([]byte(data))
}
