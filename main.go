package main

import (
	"fmt"
	"net"
	"trrt-tst/byteblaze-deamon"
	torrent_client "trrt-tst/torrent-client"

	"github.com/anacrolix/torrent"
	"github.com/sirupsen/logrus"
)

func main() {
	torrentCfg := torrent.NewDefaultClientConfig()
	torrentCfg.Seed = true

	logger := logrus.New()
	logger.SetReportCaller(true)

	config := byteblaze_deamon.Config{
		TorrentFactory: torrent_client.TorrentFactory{
			Config: *torrentCfg,
			Logger: logger,
		},
	}

	byteblazeDeamonClinet := byteblaze_deamon.NewByteBlazeDaemon(config)
	go byteblazeDeamonClinet.Start()

	_, torrentPath, dropLocation, err := byteblazeDeamonClinet.DownloadPayloadFromACoordinator()
	if err != nil {
		fmt.Println(err)
		return
	}

	byteblazeDeamonClinet.AddPeer(net.ParseIP("139.177.179.58"))
	byteblazeDeamonClinet.AddPeer(net.ParseIP("194.233.170.18"))
	byteblazeDeamonClinet.AddPeer(net.ParseIP("172.104.234.48"))

	t, err := byteblazeDeamonClinet.TorrentFactory.CreateTorrentFromFile(torrentPath)
	if err != nil {
		fmt.Println(err)
	}

	playloadForBroadcast := torrent_client.PayloadForBroadcast{
		DropLocation: dropLocation,
		Mu:           t,
	}

	fmt.Println("Downloading/seeding torrent")
	// seeding
	go byteblazeDeamonClinet.TorrentFactory.DownloadFromTorrent(playloadForBroadcast)

	fmt.Println("Broadcasting torrent")
	errors := byteblazeDeamonClinet.BroadcastTorrentFileToAllPeers(playloadForBroadcast)
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Println(err)
		}
	}
	select {}
}
