package tatar

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testDir        = filepath.Join(os.TempDir(), "tatar-test")
	testFileTar    = filepath.Join(os.TempDir(), "tatar-test.tar")
	testFileTarGz  = filepath.Join(os.TempDir(), "tatar-test.tar.gz")
	testFileTarBz2 = filepath.Join(os.TempDir(), "tatar-test.tar.bz2")
	testFileTarXz  = filepath.Join(os.TempDir(), "tatar-test.tar.xz")
	targetDir      = filepath.Join(os.TempDir(), "tatar-test-target")
	subDir         = filepath.Join(testDir, "sub")
	data1          = []byte("foobar!")
	data2          = []byte("bazinga!")
)

func init() {
	os.RemoveAll(testDir)
	os.RemoveAll(targetDir)
	os.MkdirAll(subDir, 0755)
	ioutil.WriteFile(filepath.Join(testDir, "data1.txt"), data1, 0755)
	ioutil.WriteFile(filepath.Join(subDir, "data2.txt"), data2, 0755)
}

func TestFromDirectory(t *testing.T) {
	archive, err := NewFromDirectory(testDir)
	assert.Nil(t, err)
	reader := archive.GetReader()
	assert.Nil(t, err)

	hdr, err := reader.Next()
	assert.Equal(t, "data1.txt", hdr.Name)
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, reader)
	assert.Nil(t, err)
	assert.Equal(t, data1, buf.Bytes())

	hdr, err = reader.Next()
	assert.Equal(t, "sub", hdr.Name)

	hdr, err = reader.Next()
	assert.Equal(t, "sub/data2.txt", hdr.Name)
	buf = &bytes.Buffer{}
	_, err = io.Copy(buf, reader)
	assert.Nil(t, err)
	assert.Equal(t, data2, buf.Bytes())
}

func TestToDirectory(t *testing.T) {
	archive, err := NewFromDirectory(testDir)
	assert.Nil(t, err)
	err = archive.ToDirectory(targetDir)
	assert.Nil(t, err)
	cmd := exec.Command("diff", "-r", testDir, targetDir)
	err = cmd.Run()
	assert.Nil(t, err)
}

func TestToFile(t *testing.T) {
	archive, err := NewFromDirectory(testDir)
	assert.Nil(t, err)
	archive.Compression = NO_COMPRESSION
	_, err = archive.ToFile(testFileTar)
	assert.Nil(t, err)
	archive.Compression = GZIP
	_, err = archive.ToFile(testFileTarGz)
	assert.Nil(t, err)
	archive.Compression = BZIP2
	_, err = archive.ToFile(testFileTarBz2)
	assert.Nil(t, err)
	archive.Compression = LZMA
	_, err = archive.ToFile(testFileTarXz)
	assert.Nil(t, err)
}

func TestFromFile(t *testing.T) {
	archive, err := NewFromDirectory(testDir)
	assert.Nil(t, err)
	archive.Compression = NO_COMPRESSION
	_, err = archive.ToFile(testFileTar)
	assert.Nil(t, err)
	archive.Compression = GZIP
	_, err = archive.ToFile(testFileTarGz)
	assert.Nil(t, err)
	archive.Compression = BZIP2
	_, err = archive.ToFile(testFileTarBz2)
	assert.Nil(t, err)
	archive.Compression = LZMA
	_, err = archive.ToFile(testFileTarXz)
	assert.Nil(t, err)

	restoredArchive, err := NewFromFile(testFileTar)
	assert.Nil(t, err)
	os.RemoveAll(targetDir)
	err = restoredArchive.ToDirectory(targetDir)
	cmd := exec.Command("diff", "-r", testDir, targetDir)
	err = cmd.Run()
	assert.Nil(t, err)

	restoredArchive, err = NewFromFile(testFileTarGz)
	assert.Nil(t, err)
	os.RemoveAll(targetDir)
	err = restoredArchive.ToDirectory(targetDir)
	cmd = exec.Command("diff", "-r", testDir, targetDir)
	err = cmd.Run()
	assert.Nil(t, err)

	restoredArchive, err = NewFromFile(testFileTarBz2)
	assert.Nil(t, err)
	os.RemoveAll(targetDir)
	err = restoredArchive.ToDirectory(targetDir)
	cmd = exec.Command("diff", "-r", testDir, targetDir)
	err = cmd.Run()
	assert.Nil(t, err)

	restoredArchive, err = NewFromFile(testFileTarXz)
	assert.Nil(t, err)
	os.RemoveAll(targetDir)
	err = restoredArchive.ToDirectory(targetDir)
	cmd = exec.Command("diff", "-r", testDir, targetDir)
	err = cmd.Run()
	assert.Nil(t, err)
}

func TestToData(t *testing.T) {
	archive, err := NewFromDirectory(testDir)
	assert.Nil(t, err)

	archive.Compression = NO_COMPRESSION
	data, err := archive.ToData()
	assert.Nil(t, err)
	assert.True(t, len(data) > 0)

	archive.Compression = GZIP
	data, err = archive.ToData()
	assert.Nil(t, err)
	assert.True(t, len(data) > 0)

	archive.Compression = BZIP2
	data, err = archive.ToData()
	assert.Nil(t, err)
	assert.True(t, len(data) > 0)

	archive.Compression = LZMA
	data, err = archive.ToData()
	assert.Nil(t, err)
	assert.True(t, len(data) > 0)
}

func TestFromData(t *testing.T) {

	archive, err := NewFromDirectory(testDir)
	assert.Nil(t, err)

	archive.Compression = NO_COMPRESSION
	tarData, err := archive.ToData()
	assert.Nil(t, err)
	assert.True(t, len(tarData) > 0)

	archive.Compression = GZIP
	gzData, err := archive.ToData()
	assert.Nil(t, err)
	assert.True(t, len(gzData) > 0)

	archive.Compression = BZIP2
	bz2Data, err := archive.ToData()
	assert.Nil(t, err)
	assert.True(t, len(bz2Data) > 0)

	archive.Compression = LZMA
	xzData, err := archive.ToData()
	assert.Nil(t, err)
	assert.True(t, len(xzData) > 0)

	restoredArchive, err := NewFromData(tarData, NO_COMPRESSION)
	assert.Nil(t, err)
	assert.Equal(t, archive.Data, restoredArchive.Data)

	restoredArchive, err = NewFromData(gzData, GZIP)
	assert.Nil(t, err)
	assert.Equal(t, archive.Data, restoredArchive.Data)

	restoredArchive, err = NewFromData(bz2Data, BZIP2)
	assert.Nil(t, err)
	assert.Equal(t, archive.Data, restoredArchive.Data)

	restoredArchive, err = NewFromData(xzData, LZMA)
	assert.Nil(t, err)
	assert.Equal(t, archive.Data, restoredArchive.Data)
}
