package listener

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/veggiedefender/torrent-client/bitfield"
	"github.com/veggiedefender/torrent-client/handshake"
	"github.com/veggiedefender/torrent-client/message"
	"github.com/veggiedefender/torrent-client/torrentfile"
)

func Run() {
	listen, err := net.Listen("tcp", "0.0.0.0:6881")
	if err != nil {
		log.Fatal(err)
	}
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("received connection")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	// Complete handshake
	fmt.Printf("reading handshake from %s to %s\n", conn.RemoteAddr(), conn.LocalAddr())
	hs, err := handshake.Read(conn)
	if err != nil {
		fmt.Printf("cannot read handshake %v\n", err)
		return
	}
	fmt.Printf("got action %d\n", hs.Action)

	// Read actions
	switch hs.Action {
	case handshake.HandshakeReceiveBroadcast:
		// Receive a broadcast message with the torrentfile and start downloading the file specified in the torrent file.
		fmt.Println("received broadcast and start downloading")
		tf, err := torrentfile.Open("/app/.torrent")
		if err != nil {
			fmt.Println("error while parsing .torrent")
			log.Fatal(err)
		}

		fmt.Println("starting downloading")
		err = tf.DownloadToFile("downloaded_file")
		if err != nil {
			log.Fatal(err)
		}

	case handshake.HandshakeRequest:
		// Request for a given file. When some other peer trys to download a file.
		//   - Based on the info hash identify what file does the client want
		//   - Create a map that will based on the infohash get the directory of the pieces

		// Return back the handshake
		// TODO Change this to actual info hash
		fmt.Printf("received request for a file with infohash %s", hs.InfoHash)
		req := handshake.New(handshake.HandshakeRequest, hs.InfoHash, hs.PeerID)

		_, err := conn.Write(req.Serialize())
		if err != nil {
			fmt.Println("error while sending handshake")
			return
		}
		fmt.Println("handshake completed succesfully")
		err = handleFileRequest(conn, hs.InfoHash)
		if err != nil {
			fmt.Printf("error while preocessing request: %v", err)
			return
		}
		fmt.Println("completed request")
	case handshake.HandshakeSendBroadcast:
		// Receive a message with the torrentfile, broadcast it to other peers and start download the file specified in the torrent file.
		// Load torrentfile
		tf, err := torrentfile.Open(".torrent")
		if err != nil {
			log.Fatal(err)
		}

		err = tf.DownloadToFile("downloaded_file")
		if err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Println("corrupted handshake")
	}

}

func handleFileRequest(conn net.Conn, infoHash [20]byte) error {
	// Read files from /var/byteblaze
	// Check what info hashes are in the folder
	// Check what pieces do I have
	// Compose bitfield
	var bf bitfield.Bitfield
	pieces := make(map[int][]byte)

	fmt.Printf("got info hash %s", infoHash)
	pathToPieces := fmt.Sprintf("/var/byteblaze/%s", infoHash)
	_, err := os.Stat(pathToPieces)
	if err == nil {
		fmt.Println("File exists")
	} else if os.IsNotExist(err) {
		fmt.Println("File does not exist")
		return err
	} else {
		fmt.Println("error")
		return err
	}

	dir, err := os.ReadDir(pathToPieces)
	if err != nil {
		return err
	}

	for _, file := range dir {
		i, err := strconv.Atoi(file.Name())
		if err != nil {
			return err
		}
		filePath := fmt.Sprintf("%s", file.Name())
		f, err := os.Open(filePath)
		pieces[i], err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		f.Close()
		bf = bf.SetPiece(i)
	}

	// Send bitfield
	bitFieldMsg := message.Message{
		ID:      message.MsgBitfield,
		Payload: bf,
	}
	_, err = conn.Write(bitFieldMsg.Serialize())
	if err != nil {
		fmt.Println("unable to send message")
		return err
	}

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
			fmt.Println("got unchoke message")
			msg := message.Message{
				ID: message.MsgUnchoke,
			}
			conn.Write(msg.Serialize())
		case message.MsgChoke:
			fmt.Println("got choke message")
			msg := message.Message{
				ID: message.MsgChoke,
			}
			conn.Write(msg.Serialize())
		case message.MsgInterested:
			fmt.Println("got interested message")
		case message.MsgRequest:
			fmt.Println("got request message")
			if len(msg.Payload) != 12 {
				fmt.Println("got invalid length of payload for message request")
				return err
			}
			index := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
			begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
			length := int(binary.BigEndian.Uint32(msg.Payload[8:12]))
			payload := make([]byte, 8+length)
			copy(payload[0:8], msg.Payload[0:8])
			copy(payload[8:], pieces[index][begin:])

			msgWithPiece := message.Message{
				ID:      message.MsgPiece,
				Payload: payload,
			}
			// Send back the message with the piece
			conn.Write(msgWithPiece.Serialize())
		case message.MsgHave:
			fmt.Println("Got message have")
		default:
			fmt.Println("Undefined message")
			return err
		}

	}
}
