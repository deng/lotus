package repo

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/go-datastore"
	"golang.org/x/xerrors"

	dgbadger "github.com/dgraph-io/badger/v2"
	badger "github.com/ipfs/go-ds-badger2"
	levelds "github.com/ipfs/go-ds-leveldb"
	measure "github.com/ipfs/go-ds-measure"
	sqlds "github.com/ipfs/go-ds-sql"
	pg "github.com/ipfs/go-ds-sql/postgres"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"
)

type dsCtor func(path string) (datastore.Batching, error)

var fsDatastores = map[string]dsCtor{
	"chain":    chainBadgerDs,
	"metadata": sqlDs,

	// Those need to be fast for large writes... but also need a really good GC :c
	"staging": badgerDs, // miner specific

	"client": badgerDs, // client specific
}

func chainBadgerDs(path string) (datastore.Batching, error) {
	opts := badger.DefaultOptions
	opts.GcInterval = 0 // disable GC for chain datastore

	opts.Options = dgbadger.DefaultOptions("").WithTruncate(true).
		WithValueThreshold(1 << 10)

	return badger.NewDatastore(path, &opts)
}

func badgerDs(path string) (datastore.Batching, error) {
	opts := badger.DefaultOptions
	opts.Options = dgbadger.DefaultOptions("").WithTruncate(true).
		WithValueThreshold(1 << 10)

	return badger.NewDatastore(path, &opts)
}

func levelDs(path string) (datastore.Batching, error) {
	return levelds.NewDatastore(path, &levelds.Options{
		Compression: ldbopts.NoCompression,
		NoSync:      false,
		Strict:      ldbopts.StrictAll,
	})
}

func sqlDs(path string) (datastore.Batching, error) {
	mydb, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/postgres?sslmode=disable", "postgres", "123456", "192.168.0.34"))
	if err != nil {
		return nil, err
	}
	// Implement the Queries interface for your SQL impl.
	// ...or use the provided PostgreSQL queries
	queries := pg.NewQueries("metadata")
	ds := sqlds.NewDatastore(mydb, queries)
	return ds, nil
}

func (fsr *fsLockedRepo) openDatastores() (map[string]datastore.Batching, error) {
	if err := os.MkdirAll(fsr.join(fsDatastore), 0755); err != nil {
		return nil, xerrors.Errorf("mkdir %s: %w", fsr.join(fsDatastore), err)
	}

	out := map[string]datastore.Batching{}

	for p, ctor := range fsDatastores {
		prefix := datastore.NewKey(p)

		// TODO: optimization: don't init datastores we don't need
		ds, err := ctor(fsr.join(filepath.Join(fsDatastore, p)))
		if err != nil {
			return nil, xerrors.Errorf("opening datastore %s: %w", prefix, err)
		}

		ds = measure.New("fsrepo."+p, ds)

		out[datastore.NewKey(p).String()] = ds
	}

	return out, nil
}

func (fsr *fsLockedRepo) Datastore(ns string) (datastore.Batching, error) {
	fsr.dsOnce.Do(func() {
		fsr.ds, fsr.dsErr = fsr.openDatastores()
	})

	if fsr.dsErr != nil {
		return nil, fsr.dsErr
	}
	ds, ok := fsr.ds[ns]
	if ok {
		return ds, nil
	}
	return nil, xerrors.Errorf("no such datastore: %s", ns)
}
