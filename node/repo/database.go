package repo

import (
	"database/sql"
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
)

type DBKeyStore struct {
	db *sql.DB
}

func NewDBKeyStore(db *sql.DB) (types.KeyStore, error) {
	_, err := db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key_type TEXT NOT NULL UNIQUE, private_key BYTEA)", "key_info"))
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS key_info_key_type_text_pattern_ops_idx ON %s (key_type text_pattern_ops)", "key_info"))
	if err != nil {
		return nil, err
	}
	return &DBKeyStore{
		db,
	}, nil
}

// List lists all the keys stored in the KeyStore
func (ds *DBKeyStore) List() ([]string, error) {
	var out []string
	rows, err := ds.db.Query(`select key_type from key_info`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, nil
}

// Get gets a key out of keystore and returns KeyInfo corresponding to named key
func (ds *DBKeyStore) Get(k string) (types.KeyInfo, error) {
	res := types.KeyInfo{Type: k, PrivateKey: make([]byte, 0)}
	rows, err := ds.db.Query(`select private_key where key_type = ? from key_info`, k)
	if err != nil {
		return res, err
	}
	if !rows.Next() {
		return res, types.ErrKeyInfoNotFound
	}
	if err := rows.Scan(&res.PrivateKey); err != nil {
		return res, err
	}
	return res, nil
}

// Put saves a key info under given name
func (ds *DBKeyStore) Put(k string, ki types.KeyInfo) error {
	_, err := ds.db.Exec("insert into key_info (key_type,private_key) values (?,?)", k, ki.PrivateKey)
	if err != nil {
		return err
	}
	return nil
}

// Delete removes a key from keystore
func (ds *DBKeyStore) Delete(k string) error {
	_, err := ds.db.Exec("delete from key_info where key_type = ?", k)
	if err != nil {
		return err
	}
	return nil
}
