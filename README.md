
# Byte blaze

### What messages will ByteBlaze be able to handle?

- Request for a given file. When some other peer trys to download a file.
- Receive a broadcast message with the torrentfile and start downloading the file specified in the torrent file.
- Receive a message with the torrentfile, broadcast it to other peers and start download the file specified in the torrent file.


Distributing large files accross multiple hosts.

// ByteBlaze Rozrzutnik
// - We have a file on the host
// - Split the file into pieces
// - Create hashes for each piece
// - Create a torrent file
// - Send torrent file to hosts

### Downloading process

ByteBlaze daemon
- Daemon catches the .torrentfile
- Sends http request to tracker
- Tracker gives info about all peers
- Shuffle the list of peers (so that each client requests a different peace first)
- Start connecting to peers
  - Initialize handshake to peer
  - Get bitfield (an byte array that tells us what pieces does the peer have)
  - Send Unchoke message
  - Send Interested
  - Based on the bitfield check if the peer has the piece we need
  - If yes then attempt to download the piece
  	- Send a request for the piece to the peer
  	- Process the response, possible responses: choke, unchoke, the piece, have
  - Check integrity of the downloaded piece
  - If everything is validated then send a have message to the peer
