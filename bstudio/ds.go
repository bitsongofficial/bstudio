package bstudio

import (
	"fmt"
	"github.com/dgraph-io/badger"
	"os"
)

type Ds struct {
	Db *badger.DB
}

func NewDs() *Ds {
	// Open Badger, it will be created if it doesn't exist.
	db, err := badger.Open(badger.DefaultOptions(os.ExpandEnv("$HOME/.bstudio/db")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open badger db: %v", err)
		os.Exit(1)
	}

	return &Ds{Db: db}
}

func (ds *Ds) SetAndCommit(key, val []byte) error {
	txn := ds.Db.NewTransaction(true)
	defer txn.Discard()

	if err := txn.Set(key, val); err != nil {
		return err
	}
	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

func (ds *Ds) Get(key []byte) ([]byte, error) {
	txn := ds.Db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(key)

	if err != nil && err != badger.ErrKeyNotFound {
		return []byte{}, err
	}
	var valCopy []byte

	if err == nil {
		item.Value(func(val []byte) error {
			valCopy = append([]byte{}, val...)
			return nil
		})
	}

	return valCopy, nil
}
