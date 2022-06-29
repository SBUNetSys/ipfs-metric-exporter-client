package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-msgio"
	"github.com/pkg/errors"
	"io/ioutil"
	msgTypes "ipfs-export-metric-client/msgStruct"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// tcpserver which the client subscribed to
type tcpServer struct {
	// The address of the client.
	remote net.TCPAddr

	// The TCP connection.
	conn net.Conn

	// A 4-byte, big-endian frame-delimited writer.
	writer msgio.WriteCloser

	// A 4-byte, big-endian frame-delimited reader.
	reader msgio.ReadCloser
}

// A version message, exchanged between client and server once, immediately
// after the connection is established.
type versionMessage struct {
	Version uint16 `json:"version"`
}

// The current protocol version.
const serverVersion uint16 = 3

// DB global CID pool
//var DB = make(map[cid.Cid]int)
//var DBMutex = sync.Mutex{}

func establishConnection(serverAddr string, serverPort string) (net.Conn, net.TCPAddr) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", serverAddr, serverPort))
	if err != nil {
		fmt.Printf("Error at resolving tcp address %s:%s", serverAddr, serverPort)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Printf("Error at dialing tcp address %s:%s", serverAddr, serverPort)
		os.Exit(0)
	}
	return conn, *tcpAddr
}

func handshake(c *tcpServer) error {
	// check version string
	incoming, err := c.reader.ReadMsg()
	if err != nil {
		return errors.Wrap(err, "unable to receive version message")
	}

	incomingMsg := versionMessage{}
	err = json.Unmarshal(incoming, &incomingMsg)
	if err != nil {
		return errors.Wrap(err, "unable to decode version message")
	}
	c.reader.ReleaseMsg(incoming)

	log.Printf("client %s: got version message %+v", c.remote, incomingMsg)
	if incomingMsg.Version != serverVersion {
		return fmt.Errorf("client version mismatch, expected %d, got %d", serverVersion, incomingMsg.Version)
	}

	// send server version info
	buf, err := json.Marshal(versionMessage{Version: serverVersion})
	if err != nil {
		return errors.Wrap(err, "unable to encode version message")
	}

	err = c.writer.WriteMsg(buf)
	if err != nil {
		return errors.Wrap(err, "unable to send version message")
	}
	return nil
}

/*
	process incoming information, if is bitswap, extract cid and its metadata
*/
func processTCPMessage(msg *msgTypes.IncomingTCPMessage, cidFile *os.File) error {
	if msg.Event.BitswapMessage != nil {
		// case of bitswap msg
		for _, item := range msg.Event.BitswapMessage.WantlistEntries {
			c := item.Cid
			_, err := cidFile.WriteString(fmt.Sprintf("%s\n", c))
			if err != nil {
				log.Printf("Failed saving cid %s", c)
			}
		}
	}
	return nil
}
func extractCidInfo(cid cid.Cid) error {
	// lookup
	// new cid
	//DBMutex.Lock()
	//DB[cid] = 0
	//DBMutex.Unlock()
	// start http request to tika
	res, err := http.Get(fmt.Sprintf("http://127.0.0.1:8081/ipfs/%s", cid))
	if err != nil {
		log.Printf("Error at extacting cid %s", cid)
	}
	data, err := ioutil.ReadAll(res.Body)
	// un-marshal to objct
	metaData := msgTypes.TikaResponse{}
	err = json.Unmarshal(data, &metaData)
	if err != nil {
		log.Printf("Error at unmarshal cid %s", cid)
	}
	log.Printf("CID %s type %s ", cid, metaData.ContentType)

	return nil
}

func saveMsg(out []byte, saveDir string, cidFile *os.File) error {
	// create message struct and convert byte to it
	tcpMsg := msgTypes.IncomingTCPMessage{}
	err := json.Unmarshal(out, &tcpMsg)
	if err != nil {
		log.Printf("error at decode msg %s", err)
		log.Fatalln(string(out))
	}
	// saving to dir
	fileName := saveDir + tcpMsg.Event.Timestamp.String() + ".json"
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Printf("Failed creating event file %s", tcpMsg.Event.Peer)
	}
	_, err = file.Write(out)
	if err != nil {
		log.Printf("Failed saving event file %q", tcpMsg.Event.Peer)
	}
	file.Close()
	// process after saving
	go processTCPMessage(&tcpMsg, cidFile)
	return nil
}
func main() {
	serverAddr := "130.245.145.150"
	serverPort := "4321"
	savingDir := "./data/"
	c, tcpAddr := establishConnection(serverAddr, serverPort)

	fileName := "./cids.txt"
	cidFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Printf("Failed creating cid file")
	}
	defer cidFile.Close()
	server := &tcpServer{
		remote: tcpAddr,
		conn:   c,
		writer: msgio.NewWriter(c),
		reader: msgio.NewReader(c),
	}
	// cleaning for connection
	connect := make(chan os.Signal)
	signal.Notify(connect, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-connect
		c.Close()
		os.Exit(1)
	}()

	first := true
	for {
		if first {
			log.Printf("Starting handshake with %s", server.remote.String())
			err := handshake(server)
			if err != nil {
				log.Printf("Handshake failed with server %s: %s", server.remote.String(), err)
				server.conn.Close()
				os.Exit(-1)
			}
			first = false
		}
		//buf := pool.GlobalPool.Get(1024 * 512)
		msg, err := server.reader.ReadMsg()
		if err != nil {
			log.Printf("error at read msg %s", err)
		}
		// ungzip msg
		reader := bytes.NewReader(msg)
		zr, err := gzip.NewReader(reader)
		out, err := ioutil.ReadAll(zr)
		go saveMsg(out, savingDir, cidFile)
		//log.Printf(tcpMsg.Event.Timestamp.String())

	}

}
