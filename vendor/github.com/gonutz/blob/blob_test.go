package blob_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/gonutz/blob"
)

func TestEmptyBlobJustWritesZeroHeaderLength(t *testing.T) {
	b := blob.New()
	var buf bytes.Buffer

	err := b.Write(&buf)

	if err != nil {
		t.Fatal(err)
	}
	checkBytes(t, buf.Bytes(), []byte{0, 0, 0, 0})
}

func TestOneResourceMakesOneHeaderEntry(t *testing.T) {
	b := blob.New()
	var buf bytes.Buffer

	b.Append("id", []byte{1, 2, 3})
	err := b.Write(&buf)

	if err != nil {
		t.Fatal(err)
	}
	checkBytes(t, buf.Bytes(), []byte{
		12, 0, 0, 0,
		2, 0, // "id" is 2 bytes long
		'i', 'd',
		3, 0, 0, 0, 0, 0, 0, 0, // data length
		1, 2, 3, // actual data
	})
}

func TestTwoResourcesMakeTwoEntries(t *testing.T) {
	b := blob.New()
	buffer := bytes.NewBuffer(nil)

	b.Append("id", []byte{1, 2, 3})
	b.Append("2nd", []byte{4, 5})
	err := b.Write(buffer)

	if err != nil {
		t.Fatal(err)
	}
	checkBytes(t, buffer.Bytes(), []byte{
		25, 0, 0, 0,
		2, 0, // "id" is 2 bytes long
		'i', 'd',
		3, 0, 0, 0, 0, 0, 0, 0, // data length for "id" data
		3, 0, // "2nd" is 3 bytes long
		'2', 'n', 'd',
		2, 0, 0, 0, 0, 0, 0, 0, // data length for "2nd" data
		1, 2, 3, // data for "id"
		4, 5, // data for "2nd"
	})
}

func TestZeroLengthEntryIsStillRepresentedInHeader(t *testing.T) {
	b := blob.New()
	var buf bytes.Buffer

	b.Append("id", []byte{})
	err := b.Write(&buf)

	if err != nil {
		t.Fatal(err)
	}
	checkBytes(t, buf.Bytes(), []byte{
		12, 0, 0, 0,
		2, 0, // "id" is 2 bytes long
		'i', 'd',
		0, 0, 0, 0, 0, 0, 0, 0, // data length
		// there is no data, "id" data is empty
	})
}

func TestZeroLengthEntryCanGoBetweenTwoEntries(t *testing.T) {
	b := blob.New()
	buffer := bytes.NewBuffer(nil)

	b.Append("1", []byte{1})
	b.Append("_", []byte{})
	b.Append("2", []byte{2})
	err := b.Write(buffer)

	if err != nil {
		t.Fatal(err)
	}
	checkBytes(t, buffer.Bytes(), []byte{
		33, 0, 0, 0,
		1, 0, // "1" is 1 byte long
		'1',
		1, 0, 0, 0, 0, 0, 0, 0, // data length for "1" data
		1, 0, // "_" is 1 byte long
		'_',
		0, 0, 0, 0, 0, 0, 0, 0, // data length for "_" data
		1, 0, // "2" is 1 byte long
		'2',
		1, 0, 0, 0, 0, 0, 0, 0, // data length for "2" data
		1, 2, // data
	})
}

func TestReadingEmptyBlobReturnsZeroItems(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{
		0, 0, 0, 0, // empty header, 0 length
	})

	b, err := blob.Read(buffer)

	if err != nil {
		t.Fatal(err)
	}
	if b.ItemCount() != 0 {
		t.Fatal("item count was", b.ItemCount())
	}
}

func TestReadingOneEntryBlob(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{
		12, 0, 0, 0,
		2, 0, // "id" is 2 bytes long
		'i', 'd',
		3, 0, 0, 0, 0, 0, 0, 0, // data length
		1, 2, 3, // actual data
	})

	b, err := blob.Read(buffer)

	if err != nil {
		t.Fatal(err)
	}
	if b.ItemCount() != 1 {
		t.Fatal("item count was", b.ItemCount())
	}
	// item 1
	data, found := b.GetByID("id")
	if !found {
		t.Fatal("id not found")
	}
	checkBytes(t, data, []byte{1, 2, 3})
}

func TestReadingTwoEntryBlob(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{
		25, 0, 0, 0,
		2, 0, // "id" is 2 bytes long
		'i', 'd',
		3, 0, 0, 0, 0, 0, 0, 0, // data length
		3, 0, // "2nd" is 3 bytes long
		'2', 'n', 'd',
		2, 0, 0, 0, 0, 0, 0, 0, // data length
		1, 2, 3, // data for "id"
		4, 5, // data for "2nd"
	})

	b, err := blob.Read(buffer)

	if err != nil {
		t.Fatal(err)
	}
	if b.ItemCount() != 2 {
		t.Fatal("item count was", b.ItemCount())
	}
	// item 1
	data, found := b.GetByID("id")
	if !found {
		t.Fatal("id not found")
	}
	checkBytes(t, data, []byte{1, 2, 3})
	// item 2
	data, found = b.GetByID("2nd")
	if !found {
		t.Fatal("2nd not found")
	}
	checkBytes(t, data, []byte{4, 5})
}

func TestReadingZeroLengthDataEntry(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{
		12, 0, 0, 0,
		2, 0, // "id" is 2 bytes long
		'i', 'd',
		0, 0, 0, 0, 0, 0, 0, 0, // data length
		// no data, length is 0
	})

	b, err := blob.Read(buffer)

	if err != nil {
		t.Fatal(err)
	}
	if b.ItemCount() != 1 {
		t.Fatal("item count was", b.ItemCount())
	}
	// item 1
	data, found := b.GetByID("id")
	if !found {
		t.Fatal("id not found")
	}
	checkBytes(t, data, []byte{})
}

func TestAccessFunctions(t *testing.T) {
	b := blob.New()
	b.Append("one", []byte{1, 2, 3})
	b.Append("two", []byte{4, 5})

	if b.ItemCount() != 2 {
		t.Error("item count was", b.ItemCount())
	}

	one, found := b.GetByID("one")
	if !found {
		t.Error("one not found")
	}
	checkBytes(t, one, []byte{1, 2, 3})

	two, found := b.GetByIndex(1)
	if !found {
		t.Error("two not found by index")
	}
	checkBytes(t, two, []byte{4, 5})

	if id := b.GetIDAtIndex(-1); id != "" {
		t.Error("invalid index should result in empty string but was", id)
	}
	if id := b.GetIDAtIndex(0); id != "one" {
		t.Error("expected id is one but was", id)
	}
	if id := b.GetIDAtIndex(1); id != "two" {
		t.Error("expected id is two but was", id)
	}
}

func TestWritingMaxLengthIDIsFine(t *testing.T) {
	b := blob.New()
	var id [blob.MaxIDLen]byte
	for i := range id {
		id[i] = 'a'
	}
	b.Append(string(id[:]), nil)
	var buf bytes.Buffer

	err := b.Write(&buf)

	if err != nil {
		t.Error(err)
	}
}

func TestWritingFailsIfIDIsTooLong(t *testing.T) {
	b := blob.New()
	var id [blob.MaxIDLen + 1]byte
	for i := range id {
		id[i] = 'a'
	}
	b.Append(string(id[:]), nil)
	var buf bytes.Buffer

	err := b.Write(&buf)

	if err == nil {
		t.Error("error expected but got", buf.Bytes())
	}
}

func TestOpenBlobAndReadData(t *testing.T) {
	b := blob.New()
	b.Append("one", []byte{1, 2, 3})
	b.Append("two", []byte{4, 5})
	var buf bytes.Buffer
	b.Write(&buf)
	r := bytes.NewReader(buf.Bytes())

	br, err := blob.Open(r)
	if err != nil {
		t.Fatal(err)
	}

	{
		if n := br.ItemCount(); n != 2 {
			t.Error("want 2 items but have", n)
		}
		if s := br.GetIDAtIndex(0); s != "one" {
			t.Error("want 'one' but have", s)
		}
		if s := br.GetIDAtIndex(1); s != "two" {
			t.Error("want 'two' but have", s)
		}
	}

	{
		r, found := br.GetByID("<invalid>")
		if found {
			t.Error("invalid ID was found")
		}
		if r != nil {
			t.Error("valid reader for invalid ID")
		}
	}

	{
		r, found := br.GetByIndex(-1)
		if found {
			t.Error("invalid index was found")
		}
		if r != nil {
			t.Error("valid reader for invalid index")
		}
	}

	{
		r, _ := br.GetByIndex(0)
		all, err := ioutil.ReadAll(r)
		if err != nil {
			t.Error("reading one", err)
		}
		checkBytes(t, all, []byte{1, 2, 3})
	}

	{
		r, _ := br.GetByIndex(0)
		n, err := r.Seek(0, 123)
		if n != 0 || err.Error() != "blob.reader.Seek: invalid whence" {
			t.Error(n, err)
		}
		n, err = r.Seek(-1, io.SeekStart)
		if n != 0 || err.Error() != "blob.reader.Seek: negative position" {
			t.Error(n, err)
		}
		n, err = r.Seek(100, io.SeekEnd)
		if n != 3 {
			t.Error("seeking beyond the end should clamp to end, but returned:", n)
		}
	}

	{
		one, found := br.GetByID("one")
		if !found {
			t.Error("one not found")
		}
		all, err := ioutil.ReadAll(one)
		if err != nil {
			t.Error("reading one", err)
		}
		checkBytes(t, all, []byte{1, 2, 3})
	}

	{
		two, found := br.GetByID("two")
		if !found {
			t.Error("two not found")
		}
		all, err := ioutil.ReadAll(two)
		if err != nil {
			t.Error("reading two", err)
		}
		checkBytes(t, all, []byte{4, 5})
	}

	{
		one, found := br.GetByID("one")
		if !found {
			t.Error("one not found the second time")
		}

		pos, err := one.Seek(2, io.SeekStart)
		if err != nil {
			t.Error(err)
		}
		if pos != 2 {
			t.Error("want pos from start to be 2, got", pos)
		}

		var buffer [32]byte
		n, err := one.Read(buffer[:])
		if err != nil {
			t.Error("reading last one byte", err)
		}
		if n != 1 {
			t.Error("want one last byte of one but have", n)
		}
		if buffer[0] != 3 {
			t.Error("wrong last byte in one:", buffer[:2])
		}

		pos, err = one.Seek(-1, io.SeekEnd)
		if err != nil {
			t.Error(err)
		}
		if pos != 2 {
			t.Error("want pos from end to be 2, got", pos)
		}

		pos, err = one.Seek(-1, io.SeekCurrent)
		if err != nil {
			t.Error(err)
		}
		if pos != 1 {
			t.Error("want pos from current to be 1, got", pos)
		}
		n, err = one.Read(buffer[0:1])
		if err != nil {
			t.Error("reading second one byte", err)
		}
		if n != 1 {
			t.Error("want one last byte of one but have", n)
		}
		if buffer[0] != 2 {
			t.Error("wrong last byte in one:", buffer[:1])
		}
	}
}

func TestOpenEmptyBlob(t *testing.T) {
	buffer := bytes.NewReader([]byte{
		0, 0, 0, 0, // empty header, 0 length
	})

	b, err := blob.Open(buffer)

	if err != nil {
		t.Fatal(err)
	}
	if b.ItemCount() != 0 {
		t.Fatal("item count was", b.ItemCount())
	}
}

func TestOpenBlobWithEmptyItem(t *testing.T) {
	buffer := bytes.NewReader([]byte{
		12, 0, 0, 0,
		2, 0, // "id" is 2 bytes long
		'i', 'd',
		0, 0, 0, 0, 0, 0, 0, 0, // data length
		// no data, length was 0
	})

	b, err := blob.Open(buffer)

	if err != nil {
		t.Fatal(err)
	}
	if b.ItemCount() != 1 {
		t.Fatal("item count was", b.ItemCount())
	}
	// item 1
	data, found := b.GetByID("id")
	if !found {
		t.Fatal("id not found")
	}
	all, err := ioutil.ReadAll(data)
	if err != nil {
		t.Error("cannot read data", err)
	}
	checkBytes(t, all, []byte{})
}

func TestUnknownIDsAreNotFound(t *testing.T) {
	var b blob.Blob
	data, found := b.GetByID("<invalid>")
	if found {
		t.Error("should not have been found")
	}
	if data != nil {
		t.Error("data should be nil but was", data)
	}
}

func TestInvalidIndexIsNotFound(t *testing.T) {
	var b blob.Blob
	data, found := b.GetByIndex(-1)
	if found {
		t.Error("should not have been found")
	}
	if data != nil {
		t.Error("data should be nil but was", data)
	}
}

func TestFailingWriterForwardsErrorMessage(t *testing.T) {
	b := blob.New()
	b.Append("abc", []byte("ABC"))
	b.Append("123", []byte("456"))

	for i := 0; i < 10; i++ {
		w := &failingWriter{failAtWrite: i, errMsg: fmt.Sprintf("fail on write %d", i)}
		err := b.Write(w)
		if w.hasFailed && !strings.Contains(err.Error(), w.errMsg) {
			t.Error(err, "does not contain", w.errMsg)
		}
	}
}

type failingWriter struct {
	failAtWrite int
	errMsg      string
	hasFailed   bool
}

func (w *failingWriter) Write(b []byte) (int, error) {
	w.failAtWrite--
	if w.failAtWrite < 0 {
		w.hasFailed = true
		return 0, errors.New(w.errMsg)
	}
	return len(b), nil
}

func TestFailingReaderForwardsErrorMessage(t *testing.T) {
	b := blob.New()
	b.Append("abc", []byte("ABC"))
	var buf bytes.Buffer
	b.Write(&buf)

	for i := 0; i < 10; i++ {
		r := &failingReader{
			r:          bytes.NewReader(buf.Bytes()),
			failAtRead: i,
			errMsg:     fmt.Sprintf("read %d failed", i),
		}
		b, err := blob.Read(r)
		if r.hasFailed {
			if !strings.Contains(err.Error(), r.errMsg) {
				t.Error(err, "does not contain", r.errMsg)
			}
			if b != nil {
				t.Error("valid b after error")
			}
		}
	}
}

type failingReader struct {
	r          io.Reader
	failAtRead int
	errMsg     string
	hasFailed  bool
}

func (r *failingReader) Read(b []byte) (int, error) {
	r.failAtRead--
	if r.failAtRead < 0 {
		r.hasFailed = true
		return 0, errors.New(r.errMsg)
	}
	return r.r.Read(b)
}

func TestInvalidIDLengthInHeader(t *testing.T) {
	b, err := blob.Read(bytes.NewReader([]byte{
		1, 0, 0, 0, // header has length 1
		1, // but we need to read at least a uint16 here for the first ID length
	}))
	if !strings.HasPrefix(err.Error(), "read blob header id length: ") {
		t.Error("error was:", err)
	}
	if b != nil {
		t.Error("valid b after error")
	}
}

func TestHeaderContainsWrongIDLength(t *testing.T) {
	b, err := blob.Read(bytes.NewReader([]byte{
		5, 0, 0, 0,
		5, 0, // says ID has length 5
		'A', 'B', 'C', // but it only has 3 bytes
	}))
	if err.Error() != "read blob header id: unexpected EOF" {
		t.Error("error was:", err)
	}
	if b != nil {
		t.Error("valid b after error")
	}
}

func TestDataLengthInHeaderIsIncomplete(t *testing.T) {
	b, err := blob.Read(bytes.NewReader([]byte{
		5, 0, 0, 0,
		3, 0,
		'A', 'B', 'C',
		1, 2, 3, 4, 5, // only 5 bytes, really want a uint64 here
	}))

	if !strings.HasPrefix(err.Error(), "read blob header data length: ") {
		t.Error("error was:", err)
	}
	if b != nil {
		t.Error("valid b after error")
	}
}

func TestOpenBrokenHeader(t *testing.T) {
	b, err := blob.Open(bytes.NewReader([]byte{1, 0, 0, 0, 1}))
	if err == nil {
		t.Error("error expected")
	}
	if b != nil {
		t.Error("want nil blob after error")
	}
}

func TestOpenWithBrokenSeeker(t *testing.T) {
	var buf bytes.Buffer
	blob.New().Write(&buf)

	b, err := blob.Open(&failingSeeker{ReadSeeker: bytes.NewReader(buf.Bytes())})
	if err == nil {
		t.Error("error expected")
	}
	if b != nil {
		t.Error("want nil blob after error")
	}
}

func TestOpenBlobWithSeekerFailingEventually(t *testing.T) {
	b := blob.New()
	b.Append("abc", []byte{1, 2, 3})
	var buf bytes.Buffer
	b.Write(&buf)

	br, _ := blob.Open(&failingSeeker{
		ReadSeeker: bytes.NewReader(buf.Bytes()),
		failAtSeek: 10,
	})
	r, _ := br.GetByIndex(0)

	var err error
	for i := 0; i < 11; i++ {
		r.Seek(0, io.SeekStart)
		// record only the first read error
		if _, err1 := r.Read(make([]byte, 3)); err1 != nil && err == nil {
			err = err1
		}
	}
	if err.Error() != "failingSeeker.Seek fails" {
		t.Error(err)
	}
}

type failingSeeker struct {
	io.ReadSeeker
	failAtSeek int
}

func (s *failingSeeker) Seek(offset int64, whence int) (int64, error) {
	s.failAtSeek--
	if s.failAtSeek < 0 {
		return 0, errors.New("failingSeeker.Seek fails")
	}
	return s.ReadSeeker.Seek(offset, whence)
}

func checkBytes(t *testing.T, got, want []byte) {
	if len(got) != len(want) {
		t.Fatalf("different lengths, want %v, but got %v", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("at byte %v wanted %v but got %v", i, want[i], got[i])
		}
	}
}
