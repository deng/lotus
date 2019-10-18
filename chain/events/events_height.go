package events

import (
	"sync"

	"github.com/filecoin-project/lotus/chain/types"
)

type heightEvents struct {
	lk           sync.Mutex
	tsc          *tipSetCache
	gcConfidence uint64

	ctr triggerId

	heightTriggers map[triggerId]*heightHandler

	htTriggerHeights map[triggerH][]triggerId
	htHeights        map[msgH][]triggerId
}

func (e *heightEvents) headChangeAt(rev, app []*types.TipSet) error {
	// highest tipset is always the first (see api.ReorgOps)
	newH := app[0].Height()

	for _, ts := range rev {
		// TODO: log error if h below gcconfidence
		// revert height-based triggers

		revert := func(h uint64, ts *types.TipSet) {
			for _, tid := range e.htHeights[h] {
				// don't revert if newH is above this ts
				if newH >= h {
					continue
				}

				err := e.heightTriggers[tid].revert(ts)
				if err != nil {
					log.Errorf("reverting chain trigger (@H %d): %s", h, err)
				}
			}
		}
		revert(ts.Height(), ts)

		subh := ts.Height() - 1
		for {
			cts, err := e.tsc.get(subh)
			if err != nil {
				return err
			}

			if cts != nil {
				break
			}

			revert(subh, ts)
			subh--
		}

		if err := e.tsc.revert(ts); err != nil {
			return err
		}
	}

	tail := len(app) - 1
	for i := range app {
		ts := app[tail-i]

		if err := e.tsc.add(ts); err != nil {
			return err
		}

		// height triggers

		apply := func(h uint64, ts *types.TipSet) error {
			for _, tid := range e.htTriggerHeights[h] {
				hnd := e.heightTriggers[tid]
				triggerH := h - uint64(hnd.confidence)

				incTs, err := e.tsc.getNonNull(triggerH)
				if err != nil {
					return err
				}

				if err := hnd.handle(incTs, h); err != nil {
					log.Errorf("chain trigger (@H %d, called @ %d) failed: %s", triggerH, ts.Height(), err)
				}
			}
			return nil
		}

		if err := apply(ts.Height(), ts); err != nil {
			return err
		}
		subh := ts.Height() - 1
		for {
			cts, err := e.tsc.get(subh)
			if err != nil {
				return err
			}

			if cts != nil {
				break
			}

			if err := apply(subh, ts); err != nil {
				return err
			}

			subh--
		}

	}

	return nil
}

// ChainAt invokes the specified `HeightHandler` when the chain reaches the
//  specified height+confidence threshold. If the chain is rolled-back under the
//  specified height, `RevertHandler` will be called.
//
// ts passed to handlers is the tipset at the specified, or above, if lower tipsets were null
func (e *heightEvents) ChainAt(hnd HeightHandler, rev RevertHandler, confidence int, h uint64) error {

	e.lk.Lock() // Tricky locking, check your locks if you modify this function!

	bestH := e.tsc.best().Height()

	if bestH >= h+uint64(confidence) {
		ts, err := e.tsc.getNonNull(h)
		if err != nil {
			log.Warnf("events.ChainAt: calling HandleFunc with nil tipset, not found in cache: %s", err)
		}

		e.lk.Unlock()
		if err := hnd(ts, bestH); err != nil {
			return err
		}
		e.lk.Lock()
		bestH = e.tsc.best().Height()
	}

	defer e.lk.Unlock()

	if bestH >= h+uint64(confidence)+e.gcConfidence {
		return nil
	}

	triggerAt := h + uint64(confidence)

	id := e.ctr
	e.ctr++

	e.heightTriggers[id] = &heightHandler{
		confidence: confidence,

		handle: hnd,
		revert: rev,
	}

	e.htHeights[h] = append(e.htHeights[h], id)
	e.htTriggerHeights[triggerAt] = append(e.htTriggerHeights[triggerAt], id)

	return nil
}
