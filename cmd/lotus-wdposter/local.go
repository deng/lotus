package main

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/extern/sector-storage/stores"
	"golang.org/x/xerrors"
	"os"
	"path/filepath"
)

type LocalFaultTracker struct {
	index      stores.SectorIndex
	localStore *stores.Local
}

func NewLocalFaultTracker(local *stores.Local, sindex stores.SectorIndex) *LocalFaultTracker {
	return &LocalFaultTracker{
		localStore: local,
		index:      sindex,
	}
}

// CheckProvable returns unprovable sectors
func (l *LocalFaultTracker) CheckProvable(ctx context.Context, spt abi.RegisteredSealProof, sectors []abi.SectorID) ([]abi.SectorID, error) {
	var bad []abi.SectorID

	ssize, err := spt.SectorSize()
	if err != nil {
		return nil, err
	}

	// TODO: More better checks
	for _, sector := range sectors {
		err := func() error {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			locked, err := l.index.StorageTryLock(ctx, sector, stores.FTSealed|stores.FTCache, stores.FTNone)
			if err != nil {
				return xerrors.Errorf("acquiring sector lock: %w", err)
			}

			if !locked {
				log.Warnw("CheckProvable Sector FAULT: can't acquire read lock", "sector", sector, "sealed")
				bad = append(bad, sector)
				return nil
			}

			lp, lpDone, err := l.AcquireSector(ctx, spt, sector)
			//lp, _, err := l.localStore.AcquireSector(ctx, sector, spt, stores.FTSealed|stores.FTCache, stores.FTNone, stores.PathStorage, stores.AcquireMove)
			if err != nil {
				log.Warnw("CheckProvable Sector FAULT: acquire sector in checkProvable", "sector", sector, "error", err)
				bad = append(bad, sector)
				return nil
			}

			if lp.Sealed == "" || lp.Cache == "" {
				log.Warnw("CheckProvable Sector FAULT: cache an/or sealed paths not found", "sector", sector, "sealed", lp.Sealed, "cache", lp.Cache)
				bad = append(bad, sector)
				return nil
			}

			toCheck := map[string]int64{
				lp.Sealed:                        1,
				filepath.Join(lp.Cache, "t_aux"): 0,
				filepath.Join(lp.Cache, "p_aux"): 0,
			}

			addCachePathsForSectorSize(toCheck, lp.Cache, ssize)

			for p, sz := range toCheck {
				st, err := os.Stat(p)
				if err != nil {
					log.Warnw("CheckProvable Sector FAULT: sector file stat error", "sector", sector, "sealed", lp.Sealed, "cache", lp.Cache, "file", p, "err", err)
					bad = append(bad, sector)
					return nil
				}

				if sz != 0 {
					if st.Size() != int64(ssize)*sz {
						log.Warnw("CheckProvable Sector FAULT: sector file is wrong size", "sector", sector, "sealed", lp.Sealed, "cache", lp.Cache, "file", p, "size", st.Size(), "expectSize", int64(ssize)*sz)
						bad = append(bad, sector)
						return nil
					}
				}
			}
			if lpDone != nil {
				lpDone()
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return bad, nil
}

func (l *LocalFaultTracker) AcquireSector(ctx context.Context, spt abi.RegisteredSealProof, sid abi.SectorID) (stores.SectorPaths, func(), error) {
	var (
		out     stores.SectorPaths
		err     error
		storeID stores.ID
	)
	out, _, err = l.localStore.AcquireSector(ctx, sid, spt, stores.FTSealed|stores.FTCache, stores.FTNone, stores.PathStorage, stores.AcquireMove)
	if err == nil {
		return out, nil, nil
	}

	paths, err := l.localStore.Local(ctx)
	if err != nil {
		return stores.SectorPaths{}, nil, err
	}

	found := false
	for _, path := range paths {
		if !path.CanStore {
			continue
		}
		if path.LocalPath == "" {
			continue
		}

		found = true
		storeID = path.ID

		stores.SetPathByType(&out, stores.FTSealed, filepath.Join(path.LocalPath, stores.FTSealed.String(), stores.SectorName(sid)))
		stores.SetPathByType(&out, stores.FTCache, filepath.Join(path.LocalPath, stores.FTCache.String(), stores.SectorName(sid)))
	}
	if !found {
		return stores.SectorPaths{}, nil, xerrors.New(fmt.Sprintf("don't find any sector %d", sid))
	}

	return out, func() {
		if err := l.index.StorageDeclareSector(ctx, storeID, sid, stores.FTCache, true); err != nil {
			log.Errorf("declare sector cache error: %+v", err)
		}
		if err := l.index.StorageDeclareSector(ctx, storeID, sid, stores.FTSealed, true); err != nil {
			log.Errorf("declare sector sealed error: %+v", err)
		}
	}, nil
}

func addCachePathsForSectorSize(chk map[string]int64, cacheDir string, ssize abi.SectorSize) {
	switch ssize {
	case 2 << 10:
		fallthrough
	case 8 << 20:
		fallthrough
	case 512 << 20:
		chk[filepath.Join(cacheDir, "sc-02-data-tree-r-last.dat")] = 0
	case 32 << 30:
		for i := 0; i < 8; i++ {
			chk[filepath.Join(cacheDir, fmt.Sprintf("sc-02-data-tree-r-last-%d.dat", i))] = 0
		}
	case 64 << 30:
		for i := 0; i < 16; i++ {
			chk[filepath.Join(cacheDir, fmt.Sprintf("sc-02-data-tree-r-last-%d.dat", i))] = 0
		}
	default:
		log.Warnf("not checking cache files of %s sectors for faults", ssize)
	}
}

type LocalProvider struct {
	localStore *stores.Local
	spt        abi.RegisteredSealProof
}

func NewLocalProvider(local *stores.Local, spt abi.RegisteredSealProof) *LocalProvider {
	return &LocalProvider{
		local,
		spt,
	}
}

func (l *LocalProvider) AcquireSector(ctx context.Context, sid abi.SectorID, existing stores.SectorFileType, allocate stores.SectorFileType, ptype stores.PathType) (stores.SectorPaths, func(), error) {
	out, _, err := l.localStore.AcquireSector(ctx, sid, l.spt, existing, allocate, ptype, stores.AcquireMove)
	if err != nil {
		return out, nil, err
	}

	done := func() {}

	return out, done, nil
}
