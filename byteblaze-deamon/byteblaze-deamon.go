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

	_, err = conn.Write([]byte(pd.Mu.InfoBytes))
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
		log.Fatal(err)
	}
	defer conn.Close()

	// struct form a bytes
	var payloadForBroadcast torrent_client.PayloadForBroadcast
	err = gob.NewDecoder(conn).Decode(&payloadForBroadcast)
	if err != nil {
		log.Fatal(err)
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
func (d *ByteBlazeDaemon) DownloadPayloadFromACoordinator() (string, string, string, error) {
	payload, err := d.WaitForAFileFromACoordinator()
	if err != nil {
		return "", "", "", err
	}

	err = os.MkdirAll(payload.DropLocation, 0755)
	if err != nil {
		return "", "", "", err
	}

	err = ioutil.WriteFile(payload.DropLocation+"/"+payload.FileName, payload.File, 0644)
	if err != nil {
		return "", "", "", err
	}

	err = ioutil.WriteFile(payload.DropLocation+"/"+payload.TorrentName, payload.Torrent, 0644)
	if err != nil {
		return "", "", "", err
	}

	return payload.DropLocation + "/" + payload.FileName, payload.DropLocation + "/" + payload.TorrentName, payload.DropLocation, nil
}
