package msgStruct

import (
	bsmsg "github.com/ipfs/go-bitswap/message"
	"github.com/ipfs/go-cid"
	"time"
)

/*
A structure file from https://github.com/wiberlin/ipfs-metric-exporter
used to convert subscribed bitswap information to go struct for further process
*/
type IncomingTCPMessage struct {
	// If Event is not nil, this message is a pushed event.
	Event *event `json:"event,omitempty"`
}

// A BitswapMessage is the type pushed to remote clients for recorded incoming
// Bitswap messages.

type BitswapMessage struct {
	// Wantlist entries sent with this message.
	WantlistEntries []bsmsg.Entry `json:"wantlist_entries"`

	// Whether the wantlist entries are a full new wantlist.
	FullWantList bool `json:"full_wantlist"`

	// Blocks sent with this message.
	Blocks []cid.Cid `json:"blocks"`

	// Block presence indicators sent with this message.
	BlockPresences []BlockPresence `json:"block_presences"`

	// Underlay addresses of the peer we were connected to when the message
	// was received.
	ConnectedAddresses []string `json:"connected_addresses"`
}

// A BlockPresence indicates the presence or absence of a block.
type BlockPresence struct {
	// Cid is the referenced CID.
	Cid cid.Cid `json:"cid"`

	// Type indicates the block presence type.
	Type BlockPresenceType `json:"block_presence_type"`
}

// BlockPresenceType is an enum for presence or absence notifications.
type BlockPresenceType int

// Block presence constants.
const (
	// Have indicates that the peer has the block.
	Have BlockPresenceType = 0
	// DontHave indicates that the peer does not have the block.
	DontHave BlockPresenceType = 1
)

// ConnectionEventType specifies the type of connection event.
type ConnectionEventType int

const (
	// Connected specifies that a connection was opened.
	Connected ConnectionEventType = 0
	// Disconnected specifies that a connection was closed.
	Disconnected ConnectionEventType = 1
)

// A ConnectionEvent is the type pushed to remote clients for recorded
// connection events.
type ConnectionEvent struct {
	// The multiaddress of the remote peer.
	Remote string `json:"remote"`

	// The type of this event.
	ConnectionEventType ConnectionEventType `json:"connection_event_type"`
}

// The type sent to via TCP for pushed events.
type event struct {
	// The timestamp at which the event was recorded.
	// This defines an ordering for events.
	Timestamp time.Time `json:"timestamp"`

	// Peer is a base58-encoded string representation of the peer ID.
	Peer string `json:"peer"`

	// BitswapMessage is not nil if this event is a bitswap message.
	BitswapMessage *BitswapMessage `json:"bitswap_message,omitempty"`

	// ConnectionEvent is not nil if this event is a connection event.
	ConnectionEvent *ConnectionEvent `json:"connection_event,omitempty"`
}
