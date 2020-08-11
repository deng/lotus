package journal

import (
	"sync"
	"time"

	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("journal")

var (
	// DefaultDisabledEvents lists the journal events disabled by
	// default, usually because they are considered noisy.
	DefaultDisabledEvents = DisabledEvents{
		EventType{System: "mpool", Event: "add"},
		EventType{System: "mpool", Event: "remove"},
	}
)

// DisabledEvents is the set of event types whose journaling is suppressed.
type DisabledEvents []EventType

// EventType represents the signature of an event.
type EventType struct {
	System string
	Event  string

	// enabled stores whether this event type is enabled.
	enabled bool

	// safe is a sentinel marker that's set to true if this EventType was
	// constructed correctly (via Journal#RegisterEventType).
	safe bool
}

func (et EventType) String() string {
	return et.System + ":" + et.Event
}

// Enabled returns whether this event type is enabled in the journaling
// subsystem. Users are advised to check this before actually attempting to
// add a journal entry, as it helps bypass object construction for events that
// would be discarded anyway.
//
// All event types are enabled by default, and specific event types can only
// be disabled at Journal construction time.
func (et EventType) Enabled() bool {
	return et.safe && et.enabled
}

// EventTypeFactory is a component that constructs tracked EventType tokens,
// for usage with a Journal.
type EventTypeFactory interface {
	// RegisterEventType introduces a new event type to a journal, and
	// returns an EventType token that components can later use to check whether
	// journalling for that type is enabled/suppressed, and to tag journal
	// entries appropriately.
	RegisterEventType(system, event string) EventType
}

// Journal represents an audit trail of system actions.
//
// Every entry is tagged with a timestamp, a system name, and an event name.
// The supplied data can be any type, as long as it is JSON serializable,
// including structs, map[string]interface{}, or primitive types.
//
// For cleanliness and type safety, we recommend to use typed events. See the
// *Evt struct types in this package for more info.
type Journal interface {
	EventTypeFactory

	// RecordEvent records this event to the journal. See godocs on the Journal type
	// for more info.
	RecordEvent(evtType EventType, data interface{})

	// Close closes this journal for further writing.
	Close() error
}

// Event represents a journal entry.
//
// See godocs on Journal for more information.
type Event struct {
	EventType

	Timestamp time.Time
	Data      interface{}
}

// MaybeRecordEvent is a convenience function that evaluates if the EventType is
// enabled, and if so, it calls the supplier to create the event and
// subsequently journal.RecordEvent on the provided journal to record it.
//
// It also recovers from panics raised when calling the supplier function.
//
// This is safe to call with a nil Journal, either because the value is nil,
// or because a journal obtained through NilJournal() is in use.
func MaybeRecordEvent(journal Journal, evtType EventType, supplier func() interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Warnf("recovered from panic while recording journal event; type=%s, err=%v", evtType, r)
		}
	}()

	if journal == nil || journal == nilj {
		return
	}
	if !evtType.Enabled() {
		return
	}
	journal.RecordEvent(evtType, supplier())
}

// eventTypeFactory is an embeddable mixin that takes care of tracking disabled
// event types, and returning initialized/safe EventTypes when requested.
type eventTypeFactory struct {
	sync.Mutex

	m map[string]EventType
}

var _ EventTypeFactory = (*eventTypeFactory)(nil)

func NewEventTypeFactory(disabled DisabledEvents) EventTypeFactory {
	ret := &eventTypeFactory{
		m: make(map[string]EventType, len(disabled)+32), // + extra capacity.
	}

	for _, et := range disabled {
		et.enabled, et.safe = false, true
		ret.m[et.System+":"+et.Event] = et
	}

	return ret
}

func (d *eventTypeFactory) RegisterEventType(system, event string) EventType {
	d.Lock()
	defer d.Unlock()

	key := system + ":" + event
	if et, ok := d.m[key]; ok {
		return et
	}

	et := EventType{
		System:  system,
		Event:   event,
		enabled: true,
		safe:    true,
	}

	d.m[key] = et
	return et
}
