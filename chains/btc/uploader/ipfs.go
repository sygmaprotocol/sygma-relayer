package uploader

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/ChainSafe/sygma-relayer/config/relayer"
)

const MAX_RETRIES = 3

type Uploader interface {
	Upload(proposals []map[string]interface{}) (string, error)
}

type IPFSUploader struct {
	config relayer.UploaderConfig
}

func NewIPFSUploader(config relayer.UploaderConfig) *IPFSUploader {
	return &IPFSUploader{config: config}
}

type IPFSResponse struct {
	IpfsHash string `json:"IpfsHash"`
}

func (s *IPFSUploader) Upload(dataToUpload []map[string]interface{}) (string, error) {
	jsonData, err := json.Marshal(dataToUpload)
	if err != nil {
		return "", err
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "metadata.json")
	if err != nil {
		return "", err
	}
	_, err = part.Write(jsonData)
	if err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest("POST", s.config.URL, body)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+s.config.AuthToken)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	var resp *http.Response
	for attempt := 1; attempt <= MAX_RETRIES; attempt++ {
		client := &http.Client{}
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			break
		}

		time.Sleep(time.Duration(attempt) * time.Second)
	}

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ipfsResponse IPFSResponse
	if err := json.Unmarshal(respBody, &ipfsResponse); err != nil {
		return "", err
	}

	return ipfsResponse.IpfsHash, nil
}
