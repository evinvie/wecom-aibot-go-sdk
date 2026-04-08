package wecomaibot

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
)

const chunkSize = 512 * 1024 // 512 KB per chunk

// DownloadFile fetches a media URL and decrypts it with AES-256-CBC if aesKey is provided.
func DownloadFile(url, aesKey string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read body: %w", err)
	}
	filename := filepath.Base(resp.Request.URL.Path)
	if aesKey == "" {
		return data, filename, nil
	}
	key, err := base64.StdEncoding.DecodeString(aesKey)
	if err != nil {
		return nil, "", fmt.Errorf("decode aes key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, "", fmt.Errorf("aes cipher: %w", err)
	}
	iv := key[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(data, data)
	data = pkcs7Unpad(data)
	return data, filename, nil
}

func pkcs7Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	pad := int(data[len(data)-1])
	if pad > len(data) || pad > aes.BlockSize {
		return data
	}
	return data[:len(data)-pad]
}

// UploadFile reads a local file and uploads it via the 3-step chunked upload API.
func (c *Client) UploadFile(mediaType, filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	filename := filepath.Base(filePath)
	totalChunks := int(math.Ceil(float64(len(data)) / float64(chunkSize)))
	hash := md5.Sum(data)
	md5Hex := hex.EncodeToString(hash[:])

	// Step 1: init
	initBody := UploadInitBody{
		Type: mediaType, Filename: filename,
		TotalSize: int64(len(data)), TotalChunks: totalChunks, MD5: md5Hex,
	}
	initResp, err := sendAndWait[UploadInitResp](c, CmdUploadMediaInit, initBody)
	if err != nil {
		return "", fmt.Errorf("upload init: %w", err)
	}

	// Step 2: chunks
	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := UploadChunkBody{
			UploadID: initResp.UploadID, ChunkIndex: i,
			Base64Data: base64.StdEncoding.EncodeToString(data[start:end]),
		}
		if _, err := sendAndWait[json.RawMessage](c, CmdUploadMediaChunk, chunk); err != nil {
			return "", fmt.Errorf("upload chunk %d: %w", i, err)
		}
	}

	// Step 3: finish
	finishResp, err := sendAndWait[UploadFinishResp](c, CmdUploadMediaFinish, UploadFinishBody{UploadID: initResp.UploadID})
	if err != nil {
		return "", fmt.Errorf("upload finish: %w", err)
	}
	return finishResp.MediaID, nil
}

// sendAndWait is a helper that sends a command and parses the response body.
func sendAndWait[T any](c *Client, cmd string, body any) (*T, error) {
	respFrame, err := c.Send(cmd, body)
	if err != nil {
		return nil, err
	}
	if respFrame.ErrCode != 0 {
		return nil, fmt.Errorf("server error %d: %s", respFrame.ErrCode, respFrame.ErrMsg)
	}
	var result T
	if len(respFrame.Body) > 0 {
		if err := json.Unmarshal(respFrame.Body, &result); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return &result, nil
}
