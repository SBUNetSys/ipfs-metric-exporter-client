package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/nxadm/tail"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type elasticResponse struct {
	Hit *elasticHit `json:"hits"`
}
type elasticHit struct {
	Data []*elasticData `json:"hits"`
}
type elasticData struct {
	Source *elasticSource `json:"_source"`
}
type elasticSource struct {
	MetaData *metaData `json:"metadata"`
}
type metaData struct {
	ContentType []string `json:"Content-Type"`
}

func downloadFile(cid cid.Cid, saveDir string) {
	log.Printf("Downloading cid %s", cid)
	// files that might be keys
	fileData, err := http.Get(fmt.Sprintf("http://127.0.0.1:8080/ipfs/%s", cid))
	if err != nil {
		log.Printf("Failed download cid %s", cid)
	}
	saveFile := path.Join(saveDir, cid.String())
	out, err := os.Create(saveFile)
	if err != nil {
		log.Printf("Failed create cid file %s", cid)
	}

	// Write the body to file
	_, err = io.Copy(out, fileData.Body)
	if err != nil {
		log.Printf("Failed create cid file %s", cid)
	}
	out.Close()
}
func validateCid(c cid.Cid, saveDir string) {
	jsonStr := fmt.Sprintf("{\"query\":{\"match\":{\"_id\":\"%s\"}},"+
		"\"_source\":{\"includes\":[\"metadata\"]}}", c.String())
	jsonBytes := []byte(jsonStr)
	req, err := http.NewRequest("GET", "http://127.0.0.1:9200/_search?pretty",
		bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Printf(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	// set timeout
	client.Timeout = time.Second * 15
	resp, err := client.Do(req)
	if err != nil {
		log.Printf(err.Error())
		return
	}

	body, _ := ioutil.ReadAll(resp.Body)
	elasticRes := &elasticResponse{}
	err = json.Unmarshal(body, &elasticRes)
	if err != nil {
		log.Printf("Failed unmarshal cid %s", c)
		return
	}
	if len(elasticRes.Hit.Data) > 0 && elasticRes.Hit.Data[0].Source.MetaData != nil {
		contents := elasticRes.Hit.Data[0].Source.MetaData.ContentType
		if len(contents) > 0 {
			log.Printf("CID %s type %s ", c, contents[0])
			if strings.Contains(contents[0], "text/plain;") {
				downloadFile(c, saveDir)
			}
		}
	}
	resp.Body.Close()

}
func main() {
	var logFile string
	var fileSaveDir string

	flag.StringVar(&logFile, "l", "", "Ipfs-search log to read")
	flag.StringVar(&fileSaveDir, "d", "./out", "Output path for downloaded file")
	flag.Parse()
	if logFile == "" {
		fmt.Errorf("please specifiy ipfs-search log")
		os.Exit(-1)
	}
	t, err := tail.TailFile(logFile, tail.Config{Follow: true})
	log.Printf("Started tailing file %s, saving dir %s", logFile, fileSaveDir)
	if err != nil {
		panic(err)
	}
	// create dir
	err = os.MkdirAll(fileSaveDir, os.ModePerm)
	if err != nil {
		log.Println(err)
	}

	// process log
	for line := range t.Lines {
		text := line.Text
		if strings.Contains(text, "Done crawling") {
			text = strings.ReplaceAll(text, ",", "")
			s := strings.Split(text, " ")
			var index int
			for n, val := range s {
				if val == "crawling" {
					index = n + 1
					break
				}
			}
			ipfsLink := s[index]
			ipfsLink = strings.ReplaceAll(ipfsLink, "'", "")
			cidString := strings.ReplaceAll(ipfsLink, "ipfs://", "")
			c, err := cid.Decode(cidString)
			if err != nil {
				log.Printf("Failed validate cid %s", cidString)
				continue
			}
			log.Printf("New cid discoverd %s", c)
			validateCid(c, fileSaveDir)
		}
	}

}
