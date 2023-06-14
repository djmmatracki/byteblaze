package torrent_client

import (
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/sirupsen/logrus"

	"time"
)

type TorrentClient struct {
	client *torrent.Client
	logger *logrus.Logger
}

func NewTorrentClient(config *torrent.ClientConfig, logger *logrus.Logger) *TorrentClient {
	client, err := torrent.NewClient(config)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info(client.DhtServers())

	return &TorrentClient{
		client: client,
		logger: logger,
	}
}

func (tc *TorrentClient) CreateTorrentFromFile(filePath string) (*torrent.Torrent, error) {
	torrent, err := metainfo.LoadFromFile(filePath)
	if err != nil {
		tc.logger.Errorf("Failed to load torrent file: %s", err)
		return nil, err
	}
	return tc.client.AddTorrent(torrent)
}

func (tc *TorrentClient) DownloadFromTorrent(torrentMetaData *metainfo.MetaInfo) error {
	var prevBytesCompleted int64

	torrent, err := tc.client.AddTorrent(torrentMetaData)
	if err != nil {
		tc.logger.Errorf("Failed to add torrent: %s", err)
		return err
	}
	tc.logger.Infof("Downloading torrent: %s", torrent.Name())
	<-torrent.GotInfo()

	torrent.DownloadAll()

	for {
		if torrent.BytesCompleted() >= torrent.Length() {
			tc.logger.Println("Torrent is seeding now.")
			time.Sleep(60 * time.Minute)
			break
		} else {
			stats := torrent.Stats()
			bytesCompleted := torrent.BytesCompleted()
			// Calculate download speed.
			downloadSpeed := bytesCompleted - prevBytesCompleted
			prevBytesCompleted = bytesCompleted
			tc.logger.Infof("Download speed: %d bytes/sec", downloadSpeed)
			tc.logger.Infof("Upload speed: %d bytes/sec", stats.BytesWrittenData)
			tc.logger.Infof("Still downloading, completion: %.2f%%", float64(bytesCompleted)*100/float64(torrent.Length()))
			time.Sleep(5 * time.Second)
		}
	}
	return nil
}

func (tc *TorrentClient) AddTorrentFromFile(filePath string) (*torrent.Torrent, error) {
	return tc.client.AddTorrentFromFile(filePath)
}
