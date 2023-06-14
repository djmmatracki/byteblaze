package app

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/djmmatracki/byteblaze/internal/pkg/bitfield"
	"github.com/djmmatracki/byteblaze/internal/pkg/handshake"
	"github.com/djmmatracki/byteblaze/internal/pkg/message"
	"github.com/djmmatracki/byteblaze/internal/pkg/torrentfile"
	"github.com/jackpal/bencode-go"
	"github.com/sirupsen/logrus"
)

func Run(logger *logrus.Logger, ipAddress string, port int) {
	address := fmt.Sprintf("%s:%d", ipAddress, port)
	listen, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("error while starting to listen on")
		return
	}
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			logger.Fatal(err)
		}
		logger.Println("received connection")
		go handleConnection(logger, conn)
	}
}

func handleConnection(logger *logrus.Logger, conn net.Conn) {
	defer conn.Close()
	communication := fmt.Sprintf("%s -> %s", conn.LocalAddr(), conn.RemoteAddr())
	log := logger.WithField("connection", communication).Logger

	// Complete handshake
	log.Debugf("Reading handshake from %s to %s", conn.RemoteAddr(), conn.LocalAddr())
	hs, err := handshake.Read(conn)
	if err != nil {
		log.Errorf("Cannot read handshake from %s, error: %v", conn.RemoteAddr(), err)
		return
	}

	log.Debugf("Got handshake action %d", hs.Action)
	// Read actions
	switch hs.Action {
	case handshake.HandshakeReceiveBroadcast:
		// Receive a broadcast message with the torrentfile and start downloading the file specified in the torrent file.
		log.Debugf("Received broadcast from %s, starting download", conn.RemoteAddr())
		err = receiveBroadcast(log, conn, hs)
		if err != nil {
			log.Errorf("error while receiving broadcast, error: %v", err)
			return
		}
	case handshake.HandshakeRequest:
		log.Debugf("Received request from %s for a file with infohash: '%x'", conn.RemoteAddr(), hs.InfoHash)
		err = handleFileRequest(log, conn, hs)
		if err != nil {
			log.Errorf("error while preocessing file request: %v", err)
			return
		}
		log.Debugln("Request completed")
	case handshake.HandshakeSendBroadcast:
		log.Debugln("Received send broadcast and start downloading")
		req := handshake.New(handshake.HandshakeACK, hs.InfoHash, hs.PeerID)
		_, err := conn.Write(req.Serialize())
		if err != nil {
			log.Debugln("error while sending back ACK handshake")
			return
		}

		log.Debugln("sent ACK to host")
		bto := torrentfile.BencodeTorrent{}
		err = bencode.Unmarshal(conn, &bto)
		if err != nil {
			log.Errorf("cannot unmarshal beencode file, error %v", err)
			return
		}
		log.Debugln("unmarshaling beencode file")

		tf, err := bto.ToTorrentFile()
		if err != nil {
			log.Fatal("cannot convert to TorrentFile")
			return
		}
		log.Debugln("converted bencode to torrentfile")
		log.Debugln("broadcasting messages to peers")

		// A list of peers to send broadcast
		peers := []string{
			"143.42.54.125:6881",
			"143.42.54.140:6881",
		}
		for _, peerIP := range peers {
			err = broadcastMessage(log, peerIP, bto, hs.InfoHash)
			if err != nil {
				log.Errorf("error while broadcasting message to peer %s", peerIP)
			}
		}
		log.Debugln("broadcasted messages")

		// Start downloading
		log.Debugln("starting downloading")
		err = tf.DownloadToFile(log)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Debugln("corrupted handshake")
	}

}

func handleFileRequest(logger *logrus.Logger, conn net.Conn, hs *handshake.Handshake) error {
	// Read files from /var/byteblaze
	// Check what info hashes are in the folder
	// Check what pieces do I have
	// Compose bitfield
	req := handshake.New(handshake.HandshakeACK, hs.InfoHash, hs.PeerID)

	_, err := conn.Write(req.Serialize())
	if err != nil {
		logger.Println("error while sending handshake")
		return err
	}
	logger.Println("handshake completed succesfully")

	logger.Printf("got info hash %x", hs.InfoHash)
	pathToPieces := fmt.Sprintf("/var/byteblaze/%x", hs.InfoHash)

	_, err = os.Stat(pathToPieces)
	if err == nil {
		logger.Println("File exists")
	} else if os.IsNotExist(err) {
		logger.Println("File does not exist")
		return err
	} else {
		logger.Println("error")
		return err
	}

	torrentFilePath := fmt.Sprintf("/var/byteblaze/%x/torrentfile", hs.InfoHash)
	tf, err := torrentfile.Open(torrentFilePath)
	if err != nil {
		return err
	}
	numOfPieces := len(tf.PieceHashes)

	logger.Println("reading directory pieces")
	bf, err := getBitfield(logger, pathToPieces, numOfPieces/8+1)

	// Send bitfield
	bitFieldMsg := message.Message{
		ID:      message.MsgBitfield,
		Payload: bf,
	}
	// logger.Printf("sending bitfield %x", bf)
	_, err = conn.Write(bitFieldMsg.Serialize())
	if err != nil {
		logger.Println("unable to send message")
		return err
	}
	logger.Println("sending messages")

	for {
		// Listen for messages
		msg, err := message.Read(conn)
		if err != nil {
			return err
		}

		if msg == nil {
			return err
		}

		switch msg.ID {
		case message.MsgUnchoke:
			logger.Println("got unchoke message")
			msg := message.Message{
				ID: message.MsgUnchoke,
			}
			conn.Write(msg.Serialize())
		case message.MsgChoke:
			logger.Println("got choke message")
			msg := message.Message{
				ID: message.MsgChoke,
			}
			conn.Write(msg.Serialize())
		case message.MsgInterested:
			logger.Println("got interested message")
		case message.MsgRequestBitfield:
			logger.Println("requested bitfield")
			bf, err = getBitfield(logger, pathToPieces, numOfPieces/8+1)
			if err != nil {
				return err
			}
			msg = &message.Message{
				ID:      message.MsgBitfield,
				Payload: bf,
			}
			_, err = conn.Write(msg.Serialize())
			if err != nil {
				return err
			}

		case message.MsgRequest:
			logger.Println("got request message")
			if len(msg.Payload) != 12 {
				logger.Println("got invalid length of payload for message request")
				return err
			}
			index := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
			begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
			length := int(binary.BigEndian.Uint32(msg.Payload[8:12]))
			payload := make([]byte, 8+length)
			piecePath := fmt.Sprintf("/var/byteblaze/%x/%d", hs.InfoHash, index)
			piece, err := os.ReadFile(piecePath)
			if err != nil {
				return err
			}

			copy(payload[0:8], msg.Payload[0:8])
			copy(payload[8:], piece[begin:])

			msgWithPiece := message.Message{
				ID:      message.MsgPiece,
				Payload: payload,
			}
			// Send back the message with the piece
			conn.Write(msgWithPiece.Serialize())
		case message.MsgHave:
			logger.Println("Got message have")
		default:
			logger.Println("Undefined message")
			return err
		}

	}
}

func broadcastMessage(logger *logrus.Logger, peerIP string, tf torrentfile.BencodeTorrent, infoHash [20]byte) error {
	logger.Debugln("Dialing message to peer %s", peerIP)
	conn, err := net.Dial("tcp", peerIP)
	if err != nil {
		logger.Errorf("error while dialing message to peer %s, error: %v", peerIP, err)
		return err
	}
	var peerID [20]byte
	_, err = rand.Read(peerID[:])
	if err != nil {
		logger.Errorf("error while generating peerID, error: %v", err)
		return err
	}

	req := handshake.New(handshake.HandshakeReceiveBroadcast, infoHash, peerID)
	_, err = conn.Write(req.Serialize())
	if err != nil {
		logger.Errorf("sending broadcast, error: %v", err)
		return err
	}

	_, err = handshake.Read(conn)
	if err != nil {
		logger.Errorf("cannot read ACK handshake, error: %v", err)
		return err
	}

	err = bencode.Marshal(conn, tf)
	if err != nil {
		logger.Errorf("cannot marshal torrent file to connection, error: %v", err)
		return err
	}
	defer conn.Close()
	logger.Debugf("successfuly broadcasted message to peer %s", peerIP)
	return nil
}

func createDir(logger *logrus.Logger, directoryPath string) error {
	if _, err := os.Stat(directoryPath); os.IsNotExist(err) {
		err := os.MkdirAll(directoryPath, 0755)
		if err != nil {
			logger.Errorf("Failed to create directory, error: %v", err)
			return err
		}
		logger.Debugln("Directory created:", directoryPath)
	} else {
		logger.Debugln("Directory already exists:", directoryPath)
	}
	return nil
}

func receiveBroadcast(logger *logrus.Logger, conn net.Conn, hs *handshake.Handshake) error {
	// Send back ACK handshake
	req := handshake.New(handshake.HandshakeACK, hs.InfoHash, hs.PeerID)
	_, err := conn.Write(req.Serialize())
	if err != nil {
		logger.Errorf("error while sending handshake, error: %v", err)
		return err
	}
	logger.Debugln("Successfuly sent back ACK handshake")

	// Read torrentfile
	logger.Debugln("Reading torrentfile from connection")
	bto := torrentfile.BencodeTorrent{}
	data, err := io.ReadAll(conn)
	if err != nil {
		logger.Errorln("error while reading torrentfile data")
		return err
	}
	infoHashDir := fmt.Sprintf("/var/byteblaze/%x", hs.InfoHash)
	err = createDir(logger, infoHashDir)
	if err != nil {
		logger.Errorf("error while creating directory %s", infoHashDir)
		return err
	}
	torrentFilePath := fmt.Sprintf("/var/byteblaze/%x/torrentfile", hs.InfoHash)
	err = writeToFile(torrentFilePath, data)
	if err != nil {
		logger.Errorf("error while writing torrentfile, error: %v", err)
		return err
	}
	logger.Debugf("unmarshaling torrentfile to struct, data received from connection: '%s'", data)
	buffer := bytes.NewBuffer(data)
	err = bencode.Unmarshal(buffer, &bto)
	if err != nil {
		logger.Errorf("cannot unmarshal torrentfile to go struct %v", err)
		return err
	}

	logger.Debugf("Succesfuly unmarshaled torrent file to go struct, struct: %+v", bto)
	tf, err := bto.ToTorrentFile()
	if err != nil {
		logger.Errorf("error while converting torrentfile to TorrentFile struct, error: %v")
		return err
	}

	// Start downloading
	logger.Debugf("Starting download for torrentfile %v", tf)
	err = tf.DownloadToFile(logger)
	if err != nil {
		logger.Errorf("error while downloading file, error %v", err)
		return err
	}
	return nil
}

func writeToFile(filePath string, data []byte) error {

	// Open the file in write mode, create if it doesn't exist
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		fmt.Println("Failed to write to file:", err)
		return err
	}

	fmt.Println("Data written to file:", filePath)
	return nil
}

func getBitfield(logger *logrus.Logger, pathToPieces string, bitfieldLength int) ([]byte, error) {
	dir, err := os.ReadDir(pathToPieces)
	if err != nil {
		logger.Printf("error while reading dir %s", pathToPieces)
		return nil, err
	}
	bf := make(bitfield.Bitfield, bitfieldLength)

	for _, file := range dir {
		if file.Name() == "torrentfile" {
			continue
		}
		logger.Printf("processing piece '%s'", file.Name())
		i, err := strconv.Atoi(file.Name())
		if err != nil {
			logger.Printf("error while converting file name to int: %v", err)
			return nil, err
		}
		filePath := fmt.Sprintf("%s/%s", pathToPieces, file.Name())
		logger.Printf("processing file path %s", filePath)
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		logger.Println("sucessfuly opened piece file")
		if err != nil {
			logger.Println("error while reading piece")
			return nil, err
		}
		logger.Println("succesfuly read piece")
		f.Close()
		bf.SetPiece(i)
	}
	return bf, nil
}
