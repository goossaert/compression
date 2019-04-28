package deflate

import (
    "io"
    "fmt"
)

type WriteBuffer struct {
    writer io.Writer
    buf []byte
    index int
    baseSize int
}

func NewWriteBuffer(writer io.Writer, baseSize int) *WriteBuffer {
    wb := new(WriteBuffer)
    wb.writer = writer
    wb.buf = make([]byte, baseSize*3)
    wb.baseSize = baseSize
    wb.index = 0
    return wb
}

func (wb *WriteBuffer) WriteByte(b byte) {
    wb.rotateIfNeeded()
    wb.buf[wb.index] = b
    wb.index += 1
}

func (wb *WriteBuffer) WriteBytes(source []byte) {
    i := 0;
    for i < len(source) {
        wb.rotateIfNeeded()
        step := wb.baseSize
        if i + step > len(source) {
            step = len(source) - i
        }
        copy(wb.buf[wb.index:wb.index+step], source[i:i+step])
        wb.index += step
        i += step
    }
}

func (wb *WriteBuffer) RepeatBytes(length int, distance int) {
    fmt.Printf("WB.RepeatBytes() %d %d\n", length, distance)
    wb.rotateIfNeeded()
    copy(wb.buf[wb.index:wb.index+length], wb.buf[wb.index-distance:wb.index-distance+length])
    wb.index += length
}

func (wb *WriteBuffer) Flush() (error) {
    if _, err := wb.writer.Write(wb.buf[:wb.index]); err != nil {
        return err
    }
    wb.index = 0
    return nil
}

func (wb *WriteBuffer) rotateIfNeeded() {
    if wb.index > wb.baseSize * 2 {
        if _, err := wb.writer.Write(wb.buf[:wb.baseSize]); err != nil {
            //panic
        }
        copy(wb.buf[:wb.index-wb.baseSize], wb.buf[wb.baseSize:wb.index])
        wb.index -= wb.baseSize
    }
}

