package torrent_client

import (
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/sirupsen/logrus"
	"io/ioutil"

	"os"
	"time"
)

type PayloadForBroadcast struct {
	DropLocation string
	Torrent      []byte
	TorrentName  string
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

func (tc *TorrentFactory) DownloadFromTorrent(payload PayloadForBroadcast) error {
	var prevBytesCompleted int64
	// check if path dropLocation exists
	if _, err := os.Stat(payload.DropLocation); os.IsNotExist(err) {
		os.Mkdir(payload.DropLocation, 0777)
	}

	err := ioutil.WriteFile(payload.DropLocation+"/"+payload.TorrentName, payload.Torrent, 0644)
	if err != nil {
		tc.Logger.Errorf("Failed to write torrent file: %s", err)
		return err
	}

	mu, err := tc.CreateTorrentFromFile(payload.DropLocation + "/" + payload.TorrentName)
	if err != nil {
		tc.Logger.Errorf("Failed to create torrent from file: %s", err)
		return err
	}

	tc.Config.DataDir = payload.DropLocation
	tc.Config.Seed = true
	tc.Logger.Infof("Downloading/seeding torrent to %s", payload.DropLocation)

	client := NewTorrentClient(&tc.Config)
	t, err := client.AddTorrent(mu)
	if err != nil {
		tc.Logger.Errorf("Failed to add torrent: %s", err)
		return err
	}
	tc.Logger.Infof("Downloading torrent: %s", t.Name())
	<-t.GotInfo()

	t.DownloadAll()
	for {
		if t.BytesCompleted() >= t.Length() {
			tc.Logger.Println("Torrent is seeding now.")
			time.Sleep(60 * time.Minute)
			break
		} else {
			stats := t.Stats()
			bytesCompleted := t.BytesCompleted()
			// Calculate download speed.
			downloadSpeed := bytesCompleted - prevBytesCompleted
			prevBytesCompleted = bytesCompleted
			tc.Logger.Infof("Download speed: %d bytes/sec", downloadSpeed)
			tc.Logger.Infof("Upload speed: %d bytes/sec", stats.BytesWrittenData)
			tc.Logger.Infof("Still downloading, completion: %.2f%%", float64(bytesCompleted)*100/float64(t.Length()))
			time.Sleep(5 * time.Second)
		}
	}
	return nil
}
