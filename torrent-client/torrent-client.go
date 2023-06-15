package torrent_client

import (
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/sirupsen/logrus"

	"os"
	"time"
)

type PayloadForBroadcast struct {
	DropLocation string
	Mu           *metainfo.MetaInfo
}

type TorrentFactory struct {
	Config torrent.ClientConfig
	Logger *logrus.Logger
}

func NewTorrentClient(config *torrent.ClientConfig) *torrent.Client {
	client, err := torrent.NewClient(config)
	if err != nil {
		panic(err)
	}
	return client
}

func (tc *TorrentFactory) CreateTorrentFromFile(filePath string) (*metainfo.MetaInfo, error) {
	torrent, err := metainfo.LoadFromFile(filePath)
	if err != nil {
		tc.Logger.Errorf("Failed to load torrent file: %s", err)
		return nil, err
	}
	return torrent, nil
}

func (tc *TorrentFactory) DownloadFromTorrent(torrentMetaData PayloadForBroadcast) error {
	var prevBytesCompleted int64
	err := os.MkdirAll(torrentMetaData.DropLocation, os.ModePerm)
	if err != nil {
		tc.Logger.Errorf("Failed to create directory: %s", err)
		return err
	}

	tc.Config.DataDir = torrentMetaData.DropLocation
	client := NewTorrentClient(&tc.Config)

	torrent, err := client.AddTorrent(torrentMetaData.Mu)
	if err != nil {
		tc.Logger.Errorf("Failed to add torrent: %s", err)
		return err
	}
	tc.Logger.Infof("Downloading torrent: %s", torrent.Name())
	<-torrent.GotInfo()

	torrent.DownloadAll()

	for {
		if torrent.BytesCompleted() >= torrent.Length() {
			tc.Logger.Println("Torrent is seeding now.")
			time.Sleep(60 * time.Minute)
			break
		} else {
			stats := torrent.Stats()
			bytesCompleted := torrent.BytesCompleted()
			// Calculate download speed.
			downloadSpeed := bytesCompleted - prevBytesCompleted
			prevBytesCompleted = bytesCompleted
			tc.Logger.Infof("Download speed: %d bytes/sec", downloadSpeed)
			tc.Logger.Infof("Upload speed: %d bytes/sec", stats.BytesWrittenData)
			tc.Logger.Infof("Still downloading, completion: %.2f%%", float64(bytesCompleted)*100/float64(torrent.Length()))
			time.Sleep(5 * time.Second)
		}
	}
	return nil
}
