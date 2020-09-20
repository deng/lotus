package modules

import (
	"context"
	"database/sql"

	"go.uber.org/fx"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
	sqlds "github.com/ipfs/go-ds-sql"
	pg "github.com/ipfs/go-ds-sql/postgres"
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

func DataBase(url string) dtypes.MetadataDS {
	//fmt.Sprintf("postgres://%s:%s@%s/postgres?sslmode=disable", "postgres", "123456", "192.168.0.34")
	mydb, err := sql.Open("postgres", url)
	if err != nil {
		panic(err)
	}

	_, err = mydb.Exec("CREATE TABLE IF NOT EXISTS metadata (key TEXT NOT NULL UNIQUE, data BYTEA)")
	if err != nil {
		panic(err)
	}

	_, err = mydb.Exec("CREATE INDEX IF NOT EXISTS metadata_key_text_pattern_ops_idx ON metadata (key text_pattern_ops)")
	if err != nil {
		panic(err)
	}

	// Implement the Queries interface for your SQL impl.
	// ...or use the provided PostgreSQL queries
	queries := pg.NewQueries("metadata")
	return sqlds.NewDatastore(mydb, queries)
}
