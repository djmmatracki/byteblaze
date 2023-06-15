package byteblaze_deamon

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"trrt-tst/torrent-client"
)

const (
	// Define the port on which to listen for incoming connections.
	torrentPort        = "5000"
	coordinatorPort    = "5001"
	coordiantorAddress = ""
)

type Config struct {
	TorrentFactory torrent_client.TorrentFactory
}

type Peer struct {
	IP   net.IP
	Port string
}

type ByteBlazeDaemon struct {
	mu             sync.Mutex
	peers          []Peer
	TorrentFactory torrent_client.TorrentFactory
}

func NewByteBlazeDaemon(config Config) *ByteBlazeDaemon {
	return &ByteBlazeDaemon{
		peers:          []Peer{},
		TorrentFactory: config.TorrentFactory,
	}
}

func (d *ByteBlazeDaemon) AddPeer(ip net.IP) {
	d.mu.Lock()
	d.peers = append(d.peers, Peer{IP: ip, Port: torrentPort})
	d.mu.Unlock()
}

func (d *ByteBlazeDaemon) BroadcastTorrentFileToAllPeers(pd torrent_client.PayloadForBroadcast) []error {
	var wg sync.WaitGroup
	errorsChan := make(chan error)

	for _, peer := range d.peers {
		wg.Add(1)

		go func(peer Peer, pd torrent_client.PayloadForBroadcast) {
			defer wg.Done()

			err := d.SendTorrentFileToPeer(pd, peer.IP)
			if err != nil {
				errorsChan <- err
			}
		}(peer, pd)
	}

	// Close errorsChan after all goroutines finish.
	go func() {
		wg.Wait()
		close(errorsChan)
	}()

	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}

	return errors
}

func (d *ByteBlazeDaemon) SendTorrentFileToPeer(pd torrent_client.PayloadForBroadcast, ip net.IP) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip.String(), torrentPort), 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Use gob to encode the PayloadForBroadcast object
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(pd)
	if err != nil {
		return fmt.Errorf("unable to write metainfo to connection: %w", err)
	}

	return nil
}

// TODO: Add conurency download
func (d *ByteBlazeDaemon) WaitForTorrentFromOtherPeer() torrent_client.PayloadForBroadcast {
	ln, err := net.Listen("tcp", ":"+torrentPort)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	conn, err := ln.Accept()
	if err != nil {
		log.Fatalf("unable to accept connection: %s ", err)
	}
	defer conn.Close()

	// Decode the incoming gob data to PayloadForBroadcast object
	var payloadForBroadcast torrent_client.PayloadForBroadcast
	decoder := gob.NewDecoder(conn)
	err = decoder.Decode(&payloadForBroadcast)
	if err != nil {
		log.Fatalf("decode error: %s", err)
	}

	return payloadForBroadcast
}

func (d *ByteBlazeDaemon) DownloadTorrent() error {
	mi := d.WaitForTorrentFromOtherPeer()
	return d.TorrentFactory.DownloadFromTorrent(mi)
}

func (d *ByteBlazeDaemon) GetPeers() []Peer {
	return d.peers
}

func (d *ByteBlazeDaemon) Start() {
	for {
		err := d.DownloadTorrent()
		if err != nil {
			log.Fatal(err)
		}

	}
}

type Payload struct {
	File         []byte
	Torrent      []byte
	FileName     string
	TorrentName  string
	DropLocation string
}

func (d *ByteBlazeDaemon) WaitForAFileFromACoordinator() (*Payload, error) {
	ln, err := net.Listen("tcp", ":"+coordinatorPort)
	if err != nil {
		return nil, err
	}
	defer ln.Close()

	conn, err := ln.Accept()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	payload := &Payload{}
	decoder := gob.NewDecoder(conn)
	err = decoder.Decode(payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// DownloadPayloadFromACoordinator downloads a payload from a coordinator
func (d *ByteBlazeDaemon) DownloadPayloadFromACoordinator() (*Payload, error) {
	payload, err := d.WaitForAFileFromACoordinator()
	if err != nil {
		return payload, err
	}

	err = os.MkdirAll(payload.DropLocation, 0755)
	if err != nil {
		return payload, err
	}

	err = ioutil.WriteFile(payload.DropLocation+"/"+payload.FileName, payload.File, 0644)
	if err != nil {
		return payload, err
	}

	err = ioutil.WriteFile(payload.DropLocation+"/"+payload.TorrentName, payload.Torrent, 0644)
	if err != nil {
		return payload, err
	}

	return payload, nil
}
