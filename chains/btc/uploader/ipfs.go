package uploader

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/ChainSafe/sygma-relayer/config/relayer"
)

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
	// Convert proposals to JSON
	jsonData, err := json.Marshal(dataToUpload)
	if err != nil {
		return "", err
	}

	// Create a multipart form file
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

	// Create a new request
	req, err := http.NewRequest("POST", s.config.URL, body)
	if err != nil {
		return "", err
	}

	// Set the headers
	req.Header.Add("Authorization", "Bearer "+s.config.AuthToken)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse the response
	var ipfsResponse IPFSResponse
	if err := json.Unmarshal(respBody, &ipfsResponse); err != nil {
		return "", err
	}

	return ipfsResponse.IpfsHash, nil
}
