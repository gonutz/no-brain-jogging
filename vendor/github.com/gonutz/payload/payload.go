package payload

import (
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/gonutz/osext"
)

// Read reads the whole payload at once, returning it as a byte slice.
func Read() ([]byte, error) {
	r, err := Open()
	if err != nil {
		return nil, errors.New("payload.Read: " + err.Error())
	}
	defer r.Close()

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.New("payload.Read: unable to read payload: " + err.Error())
	}
	return data, nil
}

// Open opens the payload for reading
func Open() (ReadSeekCloser, error) {
	annotate := func(msg string, err error) error {
		errMsg := "payload.Open: " + msg
		if err != nil {
			errMsg += ": " + err.Error()
		}
		return errors.New(errMsg)
	}

	// find the path currently executed file
	path, err := osext.Executable()
	if err != nil {
		return nil, annotate("unable to find executable name", err)
	}

	// The last 16 bytes in the file are the magic string "payload " folowed by
	// a uint64 that gives us the original exe's file size. This means that the
	// data starts at that offset and ends 16 bytes before the end of the file
	// (the 16 byte trailer is not part of the original data).

	file, err := os.Open(path)
	if err != nil {
		return nil, annotate("cannot open executable", err)
	}

	// the end of the data is 16 bytes before the end of the file, due to the
	// trailer
	dataEnd, err := file.Seek(-16, os.SEEK_END)
	if err != nil {
		defer file.Close()
		return nil, annotate("unable to seek to executable's end", err)
	}

	var magic [8]byte
	_, err = io.ReadFull(file, magic[:])
	if err != nil {
		defer file.Close()
		return nil, annotate("unable to read magic string", err)
	}

	if string(magic[:]) != "payload " {
		defer file.Close()
		return nil, annotate("the executable does not contain a payload", err)
	}

	var dataStart uint64
	err = binary.Read(file, binary.LittleEndian, &dataStart)
	if err != nil {
		defer file.Close()
		return nil, annotate("unable to read payload size", err)
	}

	if dataStart > uint64(dataEnd) {
		defer file.Close()
		return nil, annotate("invalid data size at file end", nil)
	}

	// go to the original exe's end, at this point the payload data starts
	_, err = file.Seek(int64(dataStart), os.SEEK_SET)
	if err != nil {
		defer file.Close()
		return nil, annotate("unable to seek to payload start", err)
	}

	return &reader{
		file:  file,
		start: int64(dataStart),
		pos:   int64(dataStart),
		end:   dataEnd,
	}, nil
}

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type reader struct {
	file            ReadSeekCloser
	start, pos, end int64
}

func (r *reader) Read(p []byte) (n int, err error) {
	if r.pos >= r.end {
		return 0, io.EOF
	}
	if int64(len(p)) > r.end-r.pos {
		p = p[:r.end-r.pos]
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
		return 0, errors.New("payload.reader.Seek: invalid whence")
	}
	if newPos < r.start {
		return r.pos - r.start, errors.New("payload.reader.Seek: negative position")
	}
	if newPos > r.end {
		newPos = r.end
	}
	var err error
	r.pos, err = r.file.Seek(newPos, io.SeekStart)
	return r.pos - r.start, err
}

func (r *reader) Close() error {
	return r.file.Close()
}
