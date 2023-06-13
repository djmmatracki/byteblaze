package handshake

import (
	"fmt"
	"io"

	"log"
)

type handshakeID uint8

const (
	HandshakeACK              handshakeID = 0
	HandshakeRequest          handshakeID = 1
	HandshakeReceiveBroadcast handshakeID = 2
	HandshakeSendBroadcast    handshakeID = 3
)

// A Handshake is a special message that a peer uses to identify itself
type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
	Action   handshakeID
}

// New creates a new handshake with the standard pstr
func New(action handshakeID, infoHash, peerID [20]byte) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
		Action:   action,
	}
}

// Serialize serializes the handshake to a buffer
func (h *Handshake) Serialize() []byte {
	buf := make([]byte, len(h.Pstr)+49)
	buf[0] = byte(len(h.Pstr))
	curr := 1
	curr += copy(buf[curr:], h.Pstr)
	curr += copy(buf[curr:], []byte{byte(h.Action)}) // 1 byte for the action that we want to perform
	curr += copy(buf[curr:], make([]byte, 7))        // 7 reserved bytes
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])
	return buf
}

// Read parses a handshake from a stream
func Read(r io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	pstrlen := int(lengthBuf[0])
	log.Printf("received pstrlen %d\n", pstrlen)

	if pstrlen == 0 {
		err := fmt.Errorf("pstrlen cannot be 0")
		return nil, err
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	log.Println("reading handshake buffer")
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		log.Println("error while reading handshake buffer")
		return nil, err
	}
	// First byte after the pstr is for the action
	log.Println("reading handshake succeded")

	action := handshakeBuf[pstrlen]
	log.Printf("got handshake action %d\n", action)
	var infoHash, peerID [20]byte

	copy(infoHash[:], handshakeBuf[pstrlen+8:pstrlen+8+20])
	copy(peerID[:], handshakeBuf[pstrlen+8+20:])

	h := Handshake{
		Pstr:     string(handshakeBuf[0:pstrlen]),
		InfoHash: infoHash,
		PeerID:   peerID,
		Action:   handshakeID(action),
	}

	return &h, nil
}
