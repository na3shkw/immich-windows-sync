package immich

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Client struct {
	ServerURL string
	APIKey    string
}

type UploadResult struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

type UploadRequest struct {
	AssetData      io.Reader
	DeviceAssetID  string
	DeviceID       string
	FileCreatedAt  string
	FileModifiedAt string
}

func (c *Client) UploadAsset(filePath string) (*UploadResult, error) {
	fileinfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	modtime := fileinfo.ModTime()
	modtimeString := modtime.Format(time.RFC3339)

	param := UploadRequest{}

	// ファイルのバイナリ
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	param.AssetData = file

	param.DeviceAssetID = filePath
	param.DeviceID = "immich-windows-sync"
	param.FileCreatedAt = modtimeString
	param.FileModifiedAt = modtimeString

	// 各フィールドへの書き込み
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("assetData", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(part, param.AssetData); err != nil {
		return nil, err
	}
	if err = writeField(writer, "deviceAssetId", param.DeviceAssetID); err != nil {
		return nil, err
	}
	if err = writeField(writer, "deviceId", param.DeviceID); err != nil {
		return nil, err
	}
	if err = writeField(writer, "fileCreatedAt", param.FileCreatedAt); err != nil {
		return nil, err
	}
	if err = writeField(writer, "fileModifiedAt", param.FileModifiedAt); err != nil {
		return nil, err
	}

	if err = writer.Close(); err != nil {
		return nil, err
	}

	// リクエストを送る
	url := c.ServerURL + "/api/assets"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 && res.StatusCode != 201 {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var data UploadResult
	if err := json.Unmarshal(resBody, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
