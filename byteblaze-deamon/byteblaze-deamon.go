package byteblaze_deamon

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"trrt-tst/torrent-client"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/sirupsen/logrus"
)

const (
	// Define the port on which to listen for incoming connections.
	torrentPort = "5000"
)

type Config struct {
	TorrentConfig *torrent.ClientConfig
}

type Peer struct {
	IP   net.IP
	Port string
}

type ByteBlazeDaemon struct {
	mu            sync.Mutex
	peers         []Peer
	TorrentClient *torrent_client.TorrentClient
}

func NewByteBlazeDaemon(config Config) *ByteBlazeDaemon {
	logger := logrus.New()

	return &ByteBlazeDaemon{
		peers:         []Peer{},
		TorrentClient: torrent_client.NewTorrentClient(config.TorrentConfig, logger),
	}
}

func (d *ByteBlazeDaemon) AddPeer(ip net.IP) {
	d.mu.Lock()
	d.peers = append(d.peers, Peer{IP: ip, Port: torrentPort})
	d.mu.Unlock()
}

func (d *ByteBlazeDaemon) BroadcastTorrentFileToAllPeers(mu metainfo.MetaInfo) []error {
	var wg sync.WaitGroup
	errorsChan := make(chan error)

	for _, peer := range d.peers {
		wg.Add(1)

		go func(peer Peer, mu metainfo.MetaInfo) {
			defer wg.Done()

			err := d.SendTorrentFileToPeer(mu, peer.IP)
			if err != nil {
				errorsChan <- err
			}
		}(peer, mu)
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

func (d *ByteBlazeDaemon) SendTorrentFileToPeer(mi metainfo.MetaInfo, ip net.IP) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip.String(), torrentPort), 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = mi.Write(conn)
	if err != nil {
		return fmt.Errorf("unable to write metainfo to connection: %w", err)
	}

	return nil
}

// TODO: Add conurency download
func (d *ByteBlazeDaemon) WaitForTorrent() *metainfo.MetaInfo {
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

	mi, err := metainfo.Load(conn)
	if err != nil {
		log.Fatal(err)
	}

	return mi
}

func (d *ByteBlazeDaemon) DownloadTorrent() error {
	mi := d.WaitForTorrent()
	return d.TorrentClient.DownloadFromTorrent(mi)
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
