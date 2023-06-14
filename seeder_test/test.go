package main

import (
	"log"
	"net"
	"os"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

func startingNodes(network string) dht.StartingNodesGetter {
	return func() ([]dht.Addr, error) {
		addr := dht.NewAddr(&net.UDPAddr{IP: net.ParseIP("172.104.234.48"), Port: 42069})
		return []dht.Addr{addr}, nil
	}
}

func main() {
	cfg := torrent.NewDefaultClientConfig()
	cfg.NoDHT = false
	cfg.DisableTrackers = true
	cfg.Seed = true
	cfg.Debug = true
	cfg.DhtStartingNodes = startingNodes

	// Set the data directory to where your file is
	cfg.DataDir = "."

	client, err := torrent.NewClient(cfg)
	if err != nil {
		log.Fatalf("error creating client: %s", err)
	}
	// Open the torrent file
	file, err := os.Open("test.torrent")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Decode the torrent file
	mi, err := metainfo.Load(file)
	if err != nil {
		log.Fatal(err)
	}

	// Add the torrent to the client
	t, err := client.AddTorrent(mi)
	if err != nil {
		log.Fatal(err)
	}

	// Wait for the torrent to be added
	<-t.GotInfo()

	// Seed the torrent
	t.DownloadAll()

	// Wait for the torrent to download
	select {}
}
