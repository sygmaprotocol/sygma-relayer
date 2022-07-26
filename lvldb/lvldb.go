package lvldb

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

type LVLDB struct {
	path string
}

func NewLvlDB(path string) (*LVLDB, error) {
	return &LVLDB{path: path}, nil
}

func (db *LVLDB) GetByKey(key []byte) ([]byte, error) {
	d, err := db.openFile()
	if err != nil {
		return nil, err
	}
	defer d.Close()

	return d.Get(key, nil)
}

func (db *LVLDB) SetByKey(key []byte, value []byte) error {
	d, err := db.openFile()
	if err != nil {
		return err
	}
	defer d.Close()

	return d.Put(key, value, nil)
}

func (db *LVLDB) openFile() (*leveldb.DB, error) {
	ldb, err := leveldb.OpenFile(db.path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "levelDB.OpenFile fail")
	}
	return ldb, nil
}
