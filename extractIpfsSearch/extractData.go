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
	MetaData  *metaData        `json:"metadata"`
	Reference []*referenceData `json:"references,omitempty"`
}
type metaData struct {
	ContentType []string `json:"Content-Type"`
}

//type reference struct {
//	ReferenceData []*referenceData
//}
type referenceData struct {
	Name string `json:"name,omitempty"`
}

func downloadFile(cid cid.Cid, saveDir string, fileName string) {
	log.Printf("Downloading cid %s, %s", cid, fileName)
	// files that might be keys
	fileData, err := http.Get(fmt.Sprintf("http://127.0.0.1:8080/ipfs/%s", cid))
	if err != nil {
		log.Printf("Failed download cid %s", cid)
	}
	saveFile := path.Join(saveDir, fileName)
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
		"\"_source\":{\"includes\":[\"metadata\", \"references\"]}}", c.String())
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
	defer resp.Body.Close()
	if err != nil {
		log.Printf(err.Error())
		return
	}

	body, _ := ioutil.ReadAll(resp.Body)
	elasticRes := &elasticResponse{}
	err = json.Unmarshal(body, &elasticRes)
	if err != nil {
		log.Printf("Failed unmarshal cid %s, %s", c, string(body))
		return
	}
	if elasticRes == nil ||
		elasticRes.Hit == nil ||
		elasticRes.Hit.Data == nil {
		log.Printf("Nil cid object %s, %s", c, string(body))
		return
	}
	if len(elasticRes.Hit.Data) > 0 && elasticRes.Hit.Data[0].Source != nil &&
		elasticRes.Hit.Data[0].Source.MetaData != nil {
		contents := elasticRes.Hit.Data[0].Source.MetaData.ContentType
		if len(contents) > 0 {
			// check file naming
			var fileName string
			if elasticRes.Hit.Data[0].Source.Reference != nil {
				fileName = elasticRes.Hit.Data[0].Source.Reference[0].Name
			} else {
				fileName = c.String()
			}
			log.Printf("CID %s type %s, %s", c, contents[0], fileName)
			if strings.Contains(contents[0], "text/plain;") {
				downloadFile(c, saveDir, fileName)
			}
		}
	}

}
func main() {
	var logFile string
	var fileSaveDir string
	log.SetOutput(os.Stdout)
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
			first := strings.Index(text, "'")
			last := strings.Index(text, "'")
			if first == -1 || last == -1 {
				log.Printf("Faild to extract %s", text)
				continue
			}
			newText := text[strings.Index(text, "'")+1 : strings.LastIndex(text, "'")]
			beginIndex := strings.Index(newText, "(")
			endIndex := strings.Index(newText, ")")

			var cidString string
			var name string
			if beginIndex == -1 || endIndex == -1 {
				// Done crawling 'ipfs://QmPAqyAfL3eG4jM7yRtMW27zrPwcdfQSxVYbGMEAgvnVLK', result: <nil>
				log.Printf("No file name extracted %s", newText)
				cidString = strings.ReplaceAll(newText, "ipfs://", "")
			} else {
				// Done crawling '108.mp4 (ipfs://QmWe67fCQNbmcfUfYNtvMMkK1VtKvsTsRMFMSjPN6SryWw)'
				name = newText[0 : beginIndex-1]
				cidString = newText[beginIndex+1 : endIndex]
				cidString = strings.ReplaceAll(cidString, "ipfs://", "")
			}

			c, err := cid.Decode(cidString)
			if err != nil {
				log.Printf("Failed validate cid %s", newText)
				continue
			}
			log.Printf("New cid discoverd %s (%s)", c, name)
			validateCid(c, fileSaveDir)
		}
	}

}
