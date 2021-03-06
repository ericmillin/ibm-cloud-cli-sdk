package configuration

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/common/file_helpers"
)

const (
	filePermissions = 0600
	dirPermissions  = 0700
)

var (
	ErrUnexpectedFileLen = errors.New("read operation returned an unexpected number of bytes")
)

type DataInterface interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

type Persistor interface {
	Exists() bool
	Load(DataInterface) error
	Save(DataInterface) error
}

type DiskPersistor struct {
	filePath string
}

func NewDiskPersistor(path string) DiskPersistor {
	return DiskPersistor{
		filePath: path,
	}
}

func (dp DiskPersistor) Exists() bool {
	return file_helpers.FileExists(dp.filePath)
}

func (dp DiskPersistor) Load(data DataInterface) error {
	err := dp.read(data)
	if os.IsPermission(err) {
		return err
	}

	if err != nil && !errors.Is(err, ErrUnexpectedFileLen) {
		err = dp.write(data)
	}

	return err
}

func (dp DiskPersistor) Save(data DataInterface) error {
	return dp.write(data)
}

func (dp DiskPersistor) read(data DataInterface) error {
	err := os.MkdirAll(filepath.Dir(dp.filePath), dirPermissions)
	if err != nil {
		return err
	}

	fi, err := os.Stat(dp.filePath)
	if err != nil {
		return []byte{}, err
	}

	bits, err := ioutil.ReadFile(dp.filePath)

	// When multiple CLI processes are running in parallel, there are
	// cases where ReadFile will return 0 bytes and no error, despite
	// the FileInfo showing size > 0. So far have not determined the
	// root cause, however it may be that another process is deleting
	// the contents, then rewriting them; i.e., Stat captures the
	// pre-rewrite state, while ReadFile captures the mid-rewrite state.
	// A locked file is likely the "real" fix.
	if fi.Size() != int64(len(bits)) {
		return ErrUnexpectedFileLen
	}

	err = data.Unmarshal(bits)
	return err
}

func (dp DiskPersistor) write(data DataInterface) error {
	bytes, err := data.Marshal()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dp.filePath, bytes, filePermissions)
	return err
}
