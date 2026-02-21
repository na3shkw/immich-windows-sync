package immich

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type Client struct {
	ServerURL string
	APIKey    string
}

type UploadResult struct {
	Id        string `json:"id"`
	Duplicate bool   `json:"duplicate"`
}

func (c *Client) UploadAsset(filePath string) (*UploadResult, error) {
	fileinfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	modtime := fileinfo.ModTime()
	modtimeString := modtime.Format(time.RFC3339)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// ファイルのバイナリ
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	part, err := writer.CreateFormFile("assetData", filePath)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(part, file); err != nil {
		return nil, err
	}

	// 任意のユニークID（ファイルパスで十分）
	if err = writeField(writer, "deviceAssetId", filePath); err != nil {
		return nil, err
	}

	// "immich-windows-sync" 固定でOK
	if err = writeField(writer, "deviceId", "immich-windows-sync"); err != nil {
		return nil, err
	}

	// fileinfo.ModTime() をISO 8601文字列に変換
	if err = writeField(writer, "fileCreatedAt", modtimeString); err != nil {
		return nil, err
	}

	// fileCreatedAtと同じ
	if err = writeField(writer, "fileModifiedAt", modtimeString); err != nil {
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
