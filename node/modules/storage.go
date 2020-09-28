package modules

import (
	"context"
	"database/sql"
	"fmt"
	sqlds "github.com/ipfs/go-ds-sql"
	pg "github.com/ipfs/go-ds-sql/postgres"

	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
)

func LockedRepo(lr repo.LockedRepo) func(lc fx.Lifecycle) repo.LockedRepo {
	return func(lc fx.Lifecycle) repo.LockedRepo {
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				return lr.Close()
			},
		})

		return lr
	}
}

func KeyStore(lr repo.LockedRepo) (types.KeyStore, error) {
	return lr.KeyStore()
}

func Datastore(r repo.LockedRepo) (dtypes.MetadataDS, error) {
	return r.Datastore("/metadata")
}

func DataBase(mydb *sql.DB) dtypes.MetadataDS {
	// Implement the Queries interface for your SQL impl.
	// ...or use the provided PostgreSQL queries
	return sqlds.NewDatastore(mydb, pg.NewQueries("metadata"))
}

func ConnetDataBase(url string) (*sql.DB, error) {
	//fmt.Sprintf("postgres://%s:%s@%s/postgres?sslmode=disable", "postgres", "123456", "192.168.0.34")
	mydb, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	_, err = mydb.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key TEXT NOT NULL UNIQUE, data BYTEA)", "metadata"))
	if err != nil {
		return nil, err
	}
	_, err = mydb.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS metadata_key_text_pattern_ops_idx ON %s (key text_pattern_ops)", "metadata"))
	if err != nil {
		return nil, err
	}
	return mydb, nil
}

func DisconnetDataBase(db *sql.DB) error {
	return db.Close()
}
