package tatar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/dsnet/compress/bzip2"
	"github.com/ulikunitz/xz"
)

// Tar contains the uncompressed tar data and the desired Compression
// This is the main struct of this packet
type Tar struct {
	Data        []byte
	Compression CompressionType
}

// CompressionType specifies the compression.
// Valid values: NO_COMPRESSION, GZIP, BZIP2, LZMA
type CompressionType int

const (
	NO_COMPRESSION CompressionType = iota
	GZIP
	BZIP2
	LZMA
)

// NewFromDirectory creates a tar archive from the contents (!) of the given directory
func NewFromDirectory(directory string) (*Tar, error) {
	directory, _ = filepath.Abs(directory)
	res := &Tar{}
	buf := &bytes.Buffer{}
	writer := tar.NewWriter(buf)
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Clean(directory) == filepath.Clean(path) {
			return nil
		}
		link := ""
		if info.Mode()&os.ModeSymlink != 0 {
			l, err := os.Readlink(path)
			if err != nil {
				return err
			}
			link = l
		}
		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}
		hdr.Name = path[len(directory)+1:]
		err = writer.WriteHeader(hdr)
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(writer, f)
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	res.Data = buf.Bytes()
	return res, nil
}

// NewFromData loades a datablob with the specified compression
func NewFromData(data []byte, compression CompressionType) (*Tar, error) {
	reader := bytes.NewReader(data)
	result := &Tar{Compression: compression}
	_, err := result.Load(reader)
	return result, err
}

// NewFromFile loades a tar from a file.
// CompressionType is guessed by fileextension
func NewFromFile(path string) (*Tar, error) {
	t := &Tar{Compression: GuessCompression(path)}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	_, err = t.Load(f)
	return t, err
}

func (t *Tar) ToData() ([]byte, error) {
	buf := &bytes.Buffer{}
	_, err := t.Save(buf)
	return buf.Bytes(), err
}

// ToFile saves the tar to a file
// If CompressionType is undefined (NO_COMPRESSION) it is guessed by fileextension
func (t *Tar) ToFile(path string) (int64, error) {
	if t.Compression == NO_COMPRESSION {
		t.Compression = GuessCompression(path)
	}
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return t.Save(f)
}

// ToDirectory extracts the tars contents into the given directory
func (t *Tar) ToDirectory(path string) error {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	return t.ForEach(func(hdr *tar.Header, reader io.Reader) error {
		if hdr.FileInfo().IsDir() {
			err := os.MkdirAll(filepath.Join(path, hdr.Name), os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
		} else if (hdr.FileInfo().Mode() & os.ModeSymlink) != 0 {
			os.Symlink(hdr.Linkname, hdr.Name)
		} else {
			targetPath := filepath.Join(path, hdr.Name)
			f, e := os.Create(targetPath)
			if e != nil {
				return e
			}
			if _, e = io.Copy(f, reader); e != nil {
				f.Close()
				return e
			}
			e = f.Chmod(os.FileMode(hdr.Mode))
			if e != nil {
				f.Close()
				return e
			}
			f.Close()
			return nil
		}
		return nil
	})
}

// Save compresses the tar into the specified writer
func (t *Tar) Save(out io.Writer) (int64, error) {
	var compressedWriter io.Writer
	switch t.Compression {
	case NO_COMPRESSION:
		compressedWriter = out
	case GZIP:
		{
			gzipWriter := gzip.NewWriter(out)
			defer gzipWriter.Close()
			compressedWriter = gzipWriter
		}
	case BZIP2:
		{
			bzip2Writer, err := bzip2.NewWriter(out, nil)
			if err != nil {
				return 0, err
			}
			defer bzip2Writer.Close()
			compressedWriter = bzip2Writer
		}
	case LZMA:
		{
			w, err := xz.NewWriter(out)
			if err != nil {
				return 0, err
			}
			defer w.Close()
			compressedWriter = w
		}
	default:
		return 0, errors.New("unknown compression")
	}
	res, err := compressedWriter.Write(t.Data)
	if err != nil {
		return 0, err
	}
	return int64(res), nil
}

// Load decompresses the tar from the specified reader
func (t *Tar) Load(in io.Reader) (int64, error) {
	var compressedReader io.Reader
	switch t.Compression {
	case NO_COMPRESSION:
		{
			compressedReader = in
		}
	case GZIP:
		{
			gzipReader, err := gzip.NewReader(in)
			if err != nil {
				return 0, err
			}
			compressedReader = gzipReader
		}
	case BZIP2:
		{
			bzip2Reader, err := bzip2.NewReader(in, nil)
			if err != nil {
				return 0, err
			}
			compressedReader = bzip2Reader
		}
	case LZMA:
		{
			r, err := xz.NewReader(in)
			if err != nil {
				return 0, err
			}
			compressedReader = r
		}
	}
	buf := &bytes.Buffer{}
	bs, err := io.Copy(buf, compressedReader)
	t.Data = buf.Bytes()
	return bs, err
}

// GetReader returns a *tar.Reader from the stdlib
func (t *Tar) GetReader() *tar.Reader {
	r := bytes.NewBuffer(t.Data)
	return tar.NewReader(r)
}

// ForEach iterates over the tars contents, and calls the given callback for each entity (directories and files)
func (t *Tar) ForEach(cb func(header *tar.Header, reader io.Reader) error) error {
	tarReader := t.GetReader()
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = cb(hdr, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}

// GuessCompression guesses the compression type by fileextension
func GuessCompression(name string) CompressionType {
	ext := filepath.Ext(name)
	switch ext {
	case ".xz", ".XZ", ".lzma", ".LZMA":
		return LZMA
	case ".bz2", ".BZ2", ".bzip2", ".BZIP2":
		return BZIP2
	case ".gz", ".GZ", ".gzip", ".GZIP":
		return GZIP
	}
	return NO_COMPRESSION
}
