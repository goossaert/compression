package huffman

import (
    "fmt"
    "io"
    "strconv"
    "bytes"
    "container/heap"

    "github.com/goossaert/compression/logging"
    "github.com/dgryski/go-bitstream"
)


type HNode struct {
    parent *HNode
    left *HNode
    right *HNode
    frequency int
    dict map[byte]bool
}

func (hnode *HNode) Byte() byte {
    var k byte
    for k = range hnode.dict {
        break
    }
    return k
}

type HTree struct {
    root *HNode
    encodedDictionary *map[byte]Transcode
}


func (hm *HTree) PrintTree(node *HNode, side string, symbols []byte) {
    var prefix bytes.Buffer
    for i := 0; i < len(symbols); i++ {
        prefix.WriteString(fmt.Sprintf(" %c  ", symbols[i]))
    }

    var chars bytes.Buffer
    for k := range node.dict {
        chars.WriteString(fmt.Sprintf("%s", string(k)))
    }

    logging.Trace.Printf("%s%s:(%s,%d)\n", prefix.String(), side, chars.String(), node.frequency)
    if node.right != nil {
        if node.left != nil {
            symbols = append(symbols, '|')
        } else {
            symbols = append(symbols, ' ')
        }
        hm.PrintTree(node.right, "R", symbols)
        symbols = symbols[:len(symbols)-1]
    }

    if node.left != nil {
        symbols = append(symbols, ' ')
        hm.PrintTree(node.left, "L", symbols)
        symbols = symbols[:len(symbols)-1]
    }
}


func (hm *HTree) Print() {
    if hm.root != nil {
        var symbols []byte
        hm.PrintTree(hm.root, "H", symbols)
    }
}


type Transcode struct {
    encoding uint32
    nbits int
}


func BuildHTree(reader io.Reader) *HTree {
    // 1. Builds frequency tables
    freqs := make(map[byte]int)
    buffer := make([]byte, 1024)
    for {
        n, err := reader.Read(buffer)
        if err == io.EOF {
            break
        }
        for i := 0 ; i < n ; i++ {
            freqs[buffer[i]] += 1
        }
    }
    logging.Trace.Printf("%v\n", freqs)

    // 2. Builds priority queue
    pq := make(PriorityQueue, len(freqs))
    i := 0
    for character, frequency := range freqs {
        dict := make(map[byte]bool)
        dict[character] = true
        node := HNode{nil, nil, nil, frequency, dict}
        pq[i] = &PQItem{
                hnode: &node,
                index: i,
        }
        i++
    }
    heap.Init(&pq)

    // 3. Builds tree based on byte frequencies
    for pq.Len() > 1 {
        item1 := heap.Pop(&pq).(*PQItem)
        item2 := heap.Pop(&pq).(*PQItem)
        dict := make(map[byte]bool)
        for k := range item1.hnode.dict {
            dict[k] = true
        }
        for k := range item2.hnode.dict {
            dict[k] = true
        }
        node := &HNode{
            nil,
            item1.hnode,
            item2.hnode,
            item1.hnode.frequency + item2.hnode.frequency,
            dict }
        item1.hnode.parent = node
        item2.hnode.parent = node

        item := &PQItem{hnode: node}
        heap.Push(&pq, item)
    }
    last := heap.Pop(&pq).(*PQItem)
    htree := HTree{last.hnode, nil}

    // 4. Creates bit encoding for every byte in the tree,
    // and stores it into a dictionary
    dictionary := make(map[byte]Transcode)
    var stack []*HNode
    stack = append(stack, htree.root)

    for len(stack) > 0 {
        indexLast := len(stack)-1
        node := stack[indexLast]
        stack = stack[:indexLast]
        if node.left != nil {
            stack = append(stack, node.left)
        }
        if node.right != nil {
            stack = append(stack, node.right)
        }
        if node.left != nil || node.right != nil {
            continue
        }

        // Walks up the parent path, and store the reverse path
        var path []uint32
        nodePath := node
        for {
            if nodePath.parent == nil {
                break
            }
            if nodePath.parent.left == nodePath {
                path = append(path, 0)
            } else {
                path = append(path, 1)
            }
            nodePath = nodePath.parent
        }

        // Transforms the path into a bit sequence, stores it
        // in the dictionary
        var encoding uint32 = 0
        for i := 0 ; i < len(path) ; i++ {
            bitPos := uint32(i)
            encoding |= path[i] << bitPos;
        }

        transcode := Transcode{
            encoding: encoding,
            nbits: len(path)}
        kbytes := node.Byte()
        dictionary[kbytes] = transcode
    }
    htree.encodedDictionary = &dictionary

    for k, v := range *htree.encodedDictionary {
        logging.Trace.Printf("%s %0*s\n", string(k), v.nbits, strconv.FormatUint(uint64(v.encoding), 2))
    }

    return &htree
}


func (htree *HTree) EncodeBytes(reader io.Reader) (encodedData *[]byte, nbits int) {
    buf := bytes.NewBuffer(nil)
    bw := bitstream.NewWriter(buf)
    nbitsWritten := 0

    buffer := make([]byte, 1024)
    for {
        n, err := reader.Read(buffer)
        if err == io.EOF {
            break
        }
        for i := 0 ; i < n ; i++ {
            if transcode, ok := (*htree.encodedDictionary)[buffer[i]] ; ok == true {
                err := bw.WriteBits(uint64(transcode.encoding), transcode.nbits)
                if err != nil {
                    fmt.Print("Unexpected error")
                }
                nbitsWritten += transcode.nbits
                logging.Trace.Printf("input %s\n", string(buffer[i]))

            } else {
                panic("error in dictionary\n")
            }
        }
    }
    bw.Flush(bitstream.Zero)

    var out []byte
    for {
        b, err := buf.ReadByte()
        if err == io.EOF {
            break
        }
        out = append(out, b)
        logging.Trace.Printf("%0*s", 8, strconv.FormatUint(uint64(b), 2))
    }
    logging.Trace.Printf("\n")

    return &out, nbitsWritten
}


func (htree *HTree) DecodeBytes(encodedData []byte, nbits int) *[]byte {
    br := bitstream.NewReader(bytes.NewReader(encodedData))
    var out []byte
    isNewChunk := true
    bitsConsumed := 0
    var node *HNode = nil
    for {
        bit, err := br.ReadBit()
        if err == io.EOF {
            break
        }
        if err != nil {
            logging.Trace.Printf("Error while reading bits\n")
            return nil
        }
        if isNewChunk {
            node = htree.root
            isNewChunk = false
        }
        if bit == bitstream.Zero {
            node = node.left
        } else {
            node = node.right
        }
        if node.left == nil && node.right == nil {
            out = append(out, node.Byte())
            logging.Trace.Printf("%s", string(node.Byte()))
            isNewChunk = true
        }

        bitsConsumed += 1
        if bitsConsumed >= nbits {
            break
        }
    }
    return &out
}

