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

// httpClient は Immich への通信に使う HTTP クライアント。
// http.DefaultClient はタイムアウトがなく、NAS のスリープやネットワーク断で応答が返らないと
// ワーカーが永久に待ち続けてしまうため、専用のクライアントでタイムアウトを設定する。
//   - ResponseHeaderTimeout: リクエストを送り切ってからレスポンスヘッダが返るまでの待ち時間
//     （接続確立・TLS ハンドシェイクのタイムアウトは DefaultTransport の設定を引き継ぐ）
//   - Timeout: 接続からレスポンス本文の読み取りまでを含めた全体の上限。
//     送信中に通信が止まった場合の最後の安全網として、写真のアップロードには十分長い値にしている
var httpClient = newHTTPClient(60*time.Second, 10*time.Minute)

func newHTTPClient(responseHeaderTimeout, timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = responseHeaderTimeout
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

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

	res, err := httpClient.Do(req)
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
