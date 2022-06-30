package main

import (
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-msgio"
	"io/ioutil"
	msgTypes "ipfs-export-metric-client/msgStruct"
	"log"
	"net"
	"net/http"
	"os"
)

func handleRequest(conn net.Conn) {
	reader := msgio.NewReader(conn)
	msg, err := reader.ReadMsg()
	log.Printf(string(msg))
	cid := string(msg)
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
	log.Printf("CID %s type %s ", cid, metaData.Metadata.ContentType)
}
func server() {
	serverAddr := "127.0.0.1"
	serverPort := "9999"
	l, err := net.Listen("tcp", serverAddr+":"+serverPort)
	if err != nil {
		log.Printf("Error at binding")
	}
	defer l.Close()
	for true {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}
