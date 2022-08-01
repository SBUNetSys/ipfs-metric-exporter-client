package metaDataExtractor

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/nxadm/tail"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

type TikaResponse struct {
	Metadata *metaData `json:"metadata"`
	Error    string    `json:"error,omitempty"`
}
type metaData struct {
	ContentType []string `json:"Content-Type"`
	//Pdf         []string `json:"pdf:PDFVersion"`
}

// DB global CID pool
var DB = make(map[cid.Cid]string)

func downloadFile(cid cid.Cid, saveDir string, gatewayUrl string) {
	log.Printf("Downloading cid %s", cid)
	// files that might be keys
	fileData, err := http.Get(fmt.Sprintf("%s/ipfs/%s", gatewayUrl, cid))
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
func extractCidInfo(cid cid.Cid, saveDir string, metaFile *os.File, tikaUrl string, gatewayUrl string) error {
	// lookup
	// new cid
	// start http request to tika
	gUrl := fmt.Sprintf("%s/ipfs/%s", gatewayUrl, cid)
	//log.Printf("Gurl %s", gUrl)
	tUrl := fmt.Sprintf("%s/extract?url=%s", tikaUrl, url.QueryEscape(gUrl))
	log.Printf("Turl %s", tUrl)
	res, err := http.Get(tUrl)
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
	metaData := TikaResponse{}
	err = json.Unmarshal(data, &metaData)
	if err != nil {
		log.Printf("Error at unmarshal cid %s", cid)
		return nil
	}
	if metaData.Error != "" {
		log.Printf("Error at tika response %s", metaData.Error)
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
		// added more range for downloading file
		if strings.Contains(val, "text") ||
			strings.Contains(val, "html") ||
			strings.Contains(val, "json") ||
			strings.Contains(val, "javascript") ||
			strings.Contains(val, "xhtml+xml") {
			go downloadFile(cid, saveDir, gatewayUrl)
		}
	} else {
		val = "null"
		log.Printf("CID %s failed identify", cid)
	}
	DB[cid] = val
	return nil
}

func MetaExtract(saveDir string, cidFile string, tikaUrl string, gatewayUrl string) {
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
			extractCidInfo(c, fileSaveDir, metaInfoFile, tikaUrl, gatewayUrl)
		}
	}
}
