package main

import (
	"net"
	"trrt-tst/byteblaze-deamon"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/torrent"
)

//	func startingNodes(network string) dht.StartingNodesGetter {
//		return func() (ret []dht.NodeAddr) {
//	}
func startingNodes(network string) dht.StartingNodesGetter {
	return func() ([]dht.Addr, error) {
		addr := dht.NewAddr(&net.UDPAddr{IP: net.ParseIP("172.104.234.48"), Port: 42069})
		return []dht.Addr{addr}, nil
	}
}

func main() {
	torrentCfg := torrent.NewDefaultClientConfig()
	torrentCfg.NoDHT = false
	torrentCfg.DisableTrackers = true
	torrentCfg.Seed = true
	torrentCfg.DhtStartingNodes = startingNodes

	config := byteblaze_deamon.Config{
		TorrentConfig: torrent.NewDefaultClientConfig(),
	}
	byteblaze_deamon := byteblaze_deamon.NewByteBlazeDaemon(config)
	byteblaze_deamon.Start()

	// t, err := byteblaze_deamon.TorrentClient.AddTorrentFromFile("test.torrent")
	// if err != nil {
	// 	panic(err)
	// }

	// byteblaze_deamon.AddPeer(net.ParseIP("139.177.179.58"))
	// byteblaze_deamon.AddPeer(net.ParseIP("194.233.170.18"))

	// metainfo := t.Metainfo()
	// errors := byteblaze_deamon.BroadcastTorrentFileToAllPeers(metainfo)
	// if len(errors) > 0 {
	// 	for _, err := range errors {
	// 		fmt.Println(err)
	// 	}
	// }
}
