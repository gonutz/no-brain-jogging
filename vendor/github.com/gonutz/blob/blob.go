package blob

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// Blob is an in-memory data buffer, matching string IDs to byte slices (blobs).
type Blob struct {
	header
	data []byte
}

type header []indexItem

type indexItem struct {
	id    string
	start uint64
	end   uint64
}

// ItemCount returns the number of blob items, i.e. pairs of string IDs and byte
// slices. When using GetIDAtIndex or GetByIndex, valid inidices range from 0 to
// ItemCount()-1.
func (h header) ItemCount() int {
	return len(h)
}

// GetIDAtIndex returns the ID of the entry at index i or the empty string if
// the given index is out of bounds. Call ItemCount for the number of items.
func (h header) GetIDAtIndex(i int) string {
	if i < 0 || i >= len(h) {
		return ""
	}
	return h[i].id
}

// New creates an empty blob. You can add data to it using Append. After adding
// all resources, you can call Write to write it to a file for example.
func New() *Blob {
	return &Blob{}
}

// Append adds the given data at the end of the blob.
func (b *Blob) Append(id string, data []byte) {
	b.header = append(
		b.header,
		indexItem{
			id,
			uint64(len(b.data)),
			uint64(len(b.data) + len(data)),
		},
	)
	b.data = append(b.data, data...)
}

// GetByID searches the blob for an entry with the given ID and returns the
// first one found. If there is no entry with the given ID, data will be nil and
// found will be false.
func (b *Blob) GetByID(id string) (data []byte, found bool) {
	for i := range b.header {
		if b.header[i].id == id {
			data = b.data[b.header[i].start:b.header[i].end]
			found = true
			return
		}
	}
	return
}

// GetByIndex returns the data of the entry at index i. If the index is out of
// bounds, data will be nil and found will be false. Call ItemCount for the
// number of items.
func (b *Blob) GetByIndex(i int) (data []byte, found bool) {
	if i < 0 || i >= len(b.header) {
		return
	}
	data = b.data[b.header[i].start:b.header[i].end]
	found = true
	return
}

var byteOrder = binary.LittleEndian

// MaxIDLen is the maximum number of bytes in an ID if you want to be able to
// Write it. If any of the IDs is longer than MaxIDLen, Write will fail.
const MaxIDLen = 65535

// Write writes the whole binary blob to the given writer. The format is as
// follows, all numbers are encoded in little endian byte order:
//
//     uint32: Header length in bytes, of the header starting after this number
//     loop, this is the header data {
//       uint16: ID length in bytes, length of the following ID
//       string: ID, UTF-8 encoded
//       uint64: data length in bytes, of the data associated with this ID
//     }
//     []byte: after the header all data is stored back-to-back
//
// Write will fail if any of the IDs has a length of more than MaxIDLen bytes,
// since this can not be represented in the above format (uint16 is used for the
// ID string's length).
//
// Note that the header does not store offsets into the data explicitly, it only
// stores the length of each item so the offset can be computed from the
// cumulative sum of all data lengths of items that come before it.
func (b *Blob) Write(w io.Writer) (err error) {
	buffer := bytes.NewBuffer(nil)
	for i := range b.header {
		// first write the ID length and then the ID
		if len(b.header[i].id) > MaxIDLen {
			return errors.New("blob.Blob.Write: ID is too long")
		}
		// writing to bytes.Buffer never returns error != nil so do not check it
		binary.Write(buffer, byteOrder, uint16(len(b.header[i].id)))
		buffer.Write([]byte(b.header[i].id))
		length := b.header[i].end - b.header[i].start
		binary.Write(buffer, byteOrder, length)
	}
	// write the header length
	err = binary.Write(w, byteOrder, uint32(buffer.Len()))
	if err != nil {
		err = errors.New("blob.Blob.Write: cannot write header length: " + err.Error())
		return
	}
	// write the actual header data
	_, err = w.Write(buffer.Bytes())
	if err != nil {
		err = errors.New("write blob header: " + err.Error())
		return
	}
	// write the data
	_, err = w.Write(b.data)
	if err != nil {
		err = errors.New("write blob data: " + err.Error())
		return
	}
	return nil
}

func readHeader(r io.Reader) (header, uint64, error) {
	// read header length
	var headerLength uint32
	err := binary.Read(r, byteOrder, &headerLength)
	if err != nil {
		return nil, 0, errors.New("read blob header length: " + err.Error())
	}

	if headerLength == 0 {
		return header{}, 0, nil
	}

	// read the actual header
	headerData := make([]byte, headerLength)
	_, err = r.Read(headerData)
	if err != nil {
		return nil, 0, errors.New("read blob header: " + err.Error())
	}

	// dissect the header, keeping track of the overall data length
	var overallDataLength uint64
	var dataLength uint64
	var idLength uint16
	headerReader := bytes.NewBuffer(headerData)
	var h header
	for headerReader.Len() > 0 {
		err = binary.Read(headerReader, byteOrder, &idLength)
		if err != nil {
			return nil, 0, errors.New("read blob header id length: " + err.Error())
		}

		id := string(headerReader.Next(int(idLength)))
		if len(id) != int(idLength) {
			return nil, 0, errors.New("read blob header id: unexpected EOF")
		}

		err = binary.Read(headerReader, byteOrder, &dataLength)
		if err != nil {
			return nil, 0, errors.New("read blob header data length: " + err.Error())
		}

		h = append(h, indexItem{
			id,
			overallDataLength,
			overallDataLength + dataLength,
		})

		overallDataLength += dataLength
	}
	return h, overallDataLength, nil
}

// Read reads a binary blob from the given reader, keeping all data in memory.
// If an error occurs, the returned blob will be nil. See Write for a
// description of the data format.
func Read(r io.Reader) (*Blob, error) {
	var b Blob
	var overallDataLength uint64
	var err error
	b.header, overallDataLength, err = readHeader(r)
	if err != nil {
		return nil, err
	}

	if overallDataLength > 0 {
		b.data = make([]byte, overallDataLength)
		_, err = io.ReadFull(r, b.data)
		if err != nil {
			return nil, errors.New("read blob data: " + err.Error())
		}
	}

	return &b, nil
}

// Open opens a blob and reads the header without reading the data (unlike Read)
// which means that the data is not kept in memory. Calling GetByID or
// GetByIndex returns a io.ReadSeeker that can be used to read the data. Note
// that all data blob readers access the same underlying io.ReadSeeker r, the
// one that you pass to Open. Thus you must be careful not to read from two
// locations in r at the same time.
//
// Example:
//     b, _ := blob.Open(file)
//     r1, _ := b.GetByIndex(0)
//     r2, _ := b.GetByIndex(1)
// In this example it is safe to read consecutively from r1 and r2, e.g. reading
// one byte from r1, then one byte from r2, then again one byte from r1, etc.
// You can not, however, read from r1 and r2 in parallel, e.g. in two different
// Go routines since the underlying io.ReadSeeker is the same for both and in
// each Read on r1 and r2, the position of r is set before reading.
func Open(r io.ReadSeeker) (*BlobReader, error) {
	var err error
	b := BlobReader{r: r}
	b.header, _, err = readHeader(r)
	if err != nil {
		return nil, err
	}

	b.zero, err = r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, errors.New("open blob: " + err.Error())
	}

	return &b, nil
}

// BlobReader is an out-of-memory data buffer, matching string IDs to byte
// slices (blobs).
type BlobReader struct {
	header
	r    io.ReadSeeker
	zero int64
}

// GetByID searches the blob for an entry with the given ID and returns the
// first one found. If there is no entry with the given ID, r will be nil and
// found will be false.
func (b *BlobReader) GetByID(id string) (r io.ReadSeeker, found bool) {
	for i := range b.header {
		if b.header[i].id == id {
			return &reader{
				file:  b.r,
				start: b.zero + int64(b.header[i].start),
				pos:   b.zero + int64(b.header[i].start),
				end:   b.zero + int64(b.header[i].end),
			}, true
		}
	}
	return nil, false
}

// GetByIndex returns the data of the entry at index i. If the index is out of
// bounds, r will be nil and found will be false. See ItemCount for the number
// of items.
func (b *BlobReader) GetByIndex(i int) (r io.ReadSeeker, found bool) {
	if i < 0 || i >= len(b.header) {
		return nil, false
	}
	return &reader{
		file:  b.r,
		start: b.zero + int64(b.header[i].start),
		pos:   b.zero + int64(b.header[i].start),
		end:   b.zero + int64(b.header[i].end),
	}, true
}

type reader struct {
	file            io.ReadSeeker
	start, pos, end int64
}

func (r *reader) Read(p []byte) (n int, err error) {
	if r.pos >= r.end {
		return 0, io.EOF
	}
	if int64(len(p)) > r.end-r.pos {
		p = p[:r.end-r.pos]
	}
	_, err = r.file.Seek(r.pos, io.SeekStart)
	if err != nil {
		return 0, err
	}
	n, err = r.file.Read(p)
	r.pos += int64(n)
	return
}

func (r *reader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = r.start + offset
	case io.SeekCurrent:
		newPos = r.pos + offset
	case io.SeekEnd:
		newPos = r.end + offset
	default:
		return 0, errors.New("blob.reader.Seek: invalid whence")
	}
	if newPos < r.start {
		return r.pos - r.start, errors.New("blob.reader.Seek: negative position")
	}
	if newPos > r.end {
		newPos = r.end
	}
	var err error
	r.pos, err = r.file.Seek(newPos, io.SeekStart)
	return r.pos - r.start, err
}
