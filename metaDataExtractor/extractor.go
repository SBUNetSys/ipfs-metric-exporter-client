package metaDataExtractor

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/nxadm/tail"
	"io"
	"io/ioutil"
	msgTypes "ipfs-export-metric-client/msgStruct"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

// DB global CID pool
var DB = make(map[cid.Cid]string)

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
func extractCidInfo(cid cid.Cid, saveDir string, metaFile *os.File) error {
	// lookup
	// new cid
	// start http request to tika
	res, err := http.Get(fmt.Sprintf("http://127.0.0.1:8081/ipfs/%s", cid))
	if err != nil {
		log.Printf("Error at extacting cid %s", cid)
		return nil
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error at reading response %s", cid)
		return nil
	}
	// un-marshal to objct
	metaData := msgTypes.TikaResponse{}
	err = json.Unmarshal(data, &metaData)
	if err != nil {
		log.Printf("Error at unmarshal cid %s", cid)
		return nil
	}
	var val string
	if len(metaData.Metadata.ContentType) > 0 {
		val = metaData.Metadata.ContentType[0]
		log.Printf("CID %s type %s ", cid, val)
		_, err := metaFile.WriteString(fmt.Sprintf("%s %s\n", cid, val))
		if err != nil {
			log.Printf("Failed save cid %s metadata", cid)
		}
		if contain := strings.Contains(val, "text/plain;"); contain {
			go downloadFile(cid, saveDir)
		}
	} else {
		val = "null"
		log.Printf("CID %s failed identify", cid)
	}
	DB[cid] = val
	return nil
}

func MetaExtract(saveDir string, cidFile string) {
	fileSaveDir := path.Join(saveDir, "downloaded")
	err := os.MkdirAll(fileSaveDir, os.ModePerm)
	if err != nil {
		log.Println(err)
	}
	// create file for saving meta info
	metaInfoFilePath := path.Join(saveDir, "cids_meta.txt")
	metaInfoFile, err := os.OpenFile(metaInfoFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0755)

	t, err := tail.TailFile(cidFile, tail.Config{Follow: true})
	log.Printf("Started tailing file %s, saving dir %s", cidFile, fileSaveDir)
	if err != nil {
		panic(err)
	}
	// process each cid
	for line := range t.Lines {
		cidText := line.Text
		c, err := cid.Decode(cidText)
		if err != nil {
			log.Printf("Error at parsing CID %s", cidText)
		}
		if _, ok := DB[c]; ok {
			continue
		} else {
			extractCidInfo(c, fileSaveDir, metaInfoFile)
		}
	}
}
