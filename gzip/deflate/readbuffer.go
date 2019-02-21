package deflate

import (
    "io"
    "errors"
)

type ReadBuffer struct {
    buf []uint8
    reader io.Reader
    numBytesLoaded int
    index int
    bitPosition int
}

func NewReadBuffer(reader io.Reader, bufferSize int) *ReadBuffer {
    rb := new(ReadBuffer)
    rb.buf = make([]uint8, bufferSize)
    rb.reader = reader
    rb.numBytesLoaded = 0
    rb.index = 0
    rb.bitPosition = 0 // position of bit inside the byte at index 'rb.index', from the MSB
    return rb
}

func (rb *ReadBuffer) LoadMoreBytes() error {
    var numBytesRemaining = rb.numBytesLoaded - rb.index
    copy(rb.buf[0:numBytesRemaining], rb.buf[rb.index:rb.index+numBytesRemaining])
    rb.numBytesLoaded = numBytesRemaining
    n, err := rb.reader.Read(rb.buf[rb.numBytesLoaded:len(rb.buf)])
    rb.numBytesLoaded += n
    rb.index = 0
    if err != nil && err != io.EOF {
        return err
    } else {
        return nil
    }
}

func (rb *ReadBuffer) ReadBit() (bool, error) {
    if rb.index >= rb.numBytesLoaded {
        return false, errors.New("Index is out of bound.")
    }
    var bit uint8 = (rb.buf[rb.index] << uint(rb.bitPosition)) & 0x80
    if rb.bitPosition < 7 {
        rb.bitPosition += 1
    } else {
        rb.index += 1
        rb.bitPosition = 0
    }
    if bit == 0 {
        return false, nil
    } else {
        return true, nil
    }
}

func (rb *ReadBuffer) Rewind(n int) error {
    bitIndex := rb.index * 8 + rb.bitPosition - n
    if bitIndex < 0 {
        return errors.New("The input argument specified a number of bits to rewind that is too large.")
    }
    rb.index = int(bitIndex / 8)
    rb.bitPosition = bitIndex % 8
    return nil
}


type Prefix struct {
    bits uint32
    index int
}

func NewPrefix() *Prefix {
    p := new(Prefix)
    p.Reset()
    return p
}

func (p *Prefix) Reset() {
    p.bits = 0
    p.index = 0
}

func (p *Prefix) ReadBit(rb *ReadBuffer) error {
    if p.index >= 32 {
        return errors.New("Prefix is full, cannot add another bit to it.")
    }

    if bit, err := rb.ReadBit(); err != nil {
        return err
    } else if bit == true {
        p.bits &= 1 << uint(31 - p.index)
    }
    p.index += 1

    return nil
}











