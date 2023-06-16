package main

import (
	"fmt"
	"github.com/anacrolix/torrent"
	"github.com/sirupsen/logrus"
	"net"
	"trrt-tst/byteblaze-deamon"
	torrent_client "trrt-tst/torrent-client"
)

func main() {
	torrentCfg := torrent.NewDefaultClientConfig()
	torrentCfg.Seed = true

	logger := logrus.New()

	config := byteblaze_deamon.Config{
		TorrentFactory: torrent_client.TorrentFactory{
			Config: *torrentCfg,
			Logger: logger,
		},
	}

	byteblazeDeamonClinet := byteblaze_deamon.NewByteBlazeDaemon(config)
	go byteblazeDeamonClinet.Start()

	for {
		payload, err := byteblazeDeamonClinet.DownloadPayloadFromACoordinator()
		if err != nil {
			fmt.Println(err)
			return
		}

		byteblazeDeamonClinet.AddPeer(net.ParseIP("143.42.54.125"))
		byteblazeDeamonClinet.AddPeer(net.ParseIP("143.42.54.140"))
		//	byteblazeDeamonClinet.AddPeer(net.ParseIP("172.104.234.48"))

		playloadForBroadcast := torrent_client.PayloadForBroadcast{
			DropLocation: payload.DropLocation,
			Torrent:      payload.Torrent,
			TorrentName:  payload.TorrentName,
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
	}
}
