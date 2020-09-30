package repo

import (
	"database/sql"
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
)

type DBKeyStore struct {
	table string
	db    *sql.DB
}

func NewDBKeyStore(db *sql.DB, miner string) (types.KeyStore, error) {
	table := "key_info_" + miner
	_, err := db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key_type TEXT NOT NULL UNIQUE, private_key BYTEA)", table))
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS key_info_key_type_text_pattern_ops_idx ON %s (key_type text_pattern_ops)", table))
	if err != nil {
		return nil, err
	}
	return &DBKeyStore{
		table,
		db,
	}, nil
}

// List lists all the keys stored in the KeyStore
func (ds *DBKeyStore) List() ([]string, error) {
	var out []string
	rows, err := ds.db.Query(fmt.Sprintf("SELECT key_type FROM %s ", ds.table))
	if err != nil {
		return nil, fmt.Errorf("db key store list err :%v", err)
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
	rows, err := ds.db.Query(fmt.Sprintf("SELECT private_key FROM %s WHERE key_type = $1", ds.table), k)
	if err != nil {
		return res, fmt.Errorf("db key store get err :%v", err)
	}
	if !rows.Next() {
		return res, types.ErrKeyInfoNotFound
	}
	if err := rows.Scan(&res.PrivateKey); err != nil {
		return res, fmt.Errorf("db key store get err :%v", err)
	}
	return res, nil
}

// Put saves a key info under given name
func (ds *DBKeyStore) Put(k string, ki types.KeyInfo) error {
	_, err := ds.db.Exec(fmt.Sprintf("INSERT INTO %s (key_type, private_key) VALUES ($1, $2) ON CONFLICT (key_type) DO UPDATE SET private_key = $2", ds.table), k, ki.PrivateKey)
	if err != nil {
		return fmt.Errorf("db key store put err :%v", err)
	}
	return nil
}

// Delete removes a key from keystore
func (ds *DBKeyStore) Delete(k string) error {
	_, err := ds.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE key_type = $1", ds.table), k)
	if err != nil {
		return fmt.Errorf("db key store delete err :%v", err)
	}
	return nil
}
