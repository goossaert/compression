package deflate

import (
    "io"
)

type WriteBuffer struct {
    buf []byte
    writer io.Writer
    index int
    baseSize int
}

func NewWriteBuffer(writer io.Writer, baseSize int) *WriteBuffer {
    wb := new(WriteBuffer)
    wb.buf = make([]byte, baseSize*3)
    wb.baseSize = baseSize
    wb.index = 0
    wb.writer = writer
    return wb
}

func (wb *WriteBuffer) WriteByte(b byte) {
    wb.rotateIfNeeded()
    wb.buf[wb.index] = b
    wb.index += 1
}

func (wb *WriteBuffer) RepeatBytes(length int, distance int) {
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
        copy(wb.buf[:wb.baseSize*2], wb.buf[wb.baseSize:wb.baseSize*3])
        wb.index -= wb.baseSize
    }
}

