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

const chunkSize = 512 * 1024 // 每个分片 512 KB

// DownloadFile 下载媒体文件，若提供 aesKey 则使用 AES-256-CBC 解密。
// 返回文件内容、文件名和错误。
func DownloadFile(url, aesKey string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取响应体失败: %w", err)
	}
	filename := filepath.Base(resp.Request.URL.Path)
	if aesKey == "" {
		return data, filename, nil
	}
	key, err := base64.StdEncoding.DecodeString(aesKey)
	if err != nil {
		return nil, "", fmt.Errorf("解码 AES 密钥失败: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, "", fmt.Errorf("创建 AES 密码器失败: %w", err)
	}
	iv := key[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(data, data)
	data = pkcs7Unpad(data)
	return data, filename, nil
}

// pkcs7Unpad 移除 PKCS#7 填充。
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

// UploadFile 读取本地文件并通过三步分片上传 API 上传，返回 media_id。
func (c *Client) UploadFile(mediaType, filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	filename := filepath.Base(filePath)
	totalChunks := int(math.Ceil(float64(len(data)) / float64(chunkSize)))
	hash := md5.Sum(data)
	md5Hex := hex.EncodeToString(hash[:])

	// 第一步：初始化上传
	initBody := UploadInitBody{
		Type: mediaType, Filename: filename,
		TotalSize: int64(len(data)), TotalChunks: totalChunks, MD5: md5Hex,
	}
	initResp, err := sendAndWait[UploadInitResp](c, CmdUploadMediaInit, initBody)
	if err != nil {
		return "", fmt.Errorf("上传初始化失败: %w", err)
	}

	// 第二步：逐个上传分片
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
			return "", fmt.Errorf("上传第 %d 个分片失败: %w", i, err)
		}
	}

	// 第三步：完成上传
	finishResp, err := sendAndWait[UploadFinishResp](c, CmdUploadMediaFinish, UploadFinishBody{UploadID: initResp.UploadID})
	if err != nil {
		return "", fmt.Errorf("完成上传失败: %w", err)
	}
	return finishResp.MediaID, nil
}

// sendAndWait 发送命令并解析响应体的泛型辅助函数。
func sendAndWait[T any](c *Client, cmd string, body any) (*T, error) {
	respFrame, err := c.Send(cmd, body)
	if err != nil {
		return nil, err
	}
	if respFrame.ErrCode != 0 {
		return nil, fmt.Errorf("服务端错误 %d: %s", respFrame.ErrCode, respFrame.ErrMsg)
	}
	var result T
	if len(respFrame.Body) > 0 {
		if err := json.Unmarshal(respFrame.Body, &result); err != nil {
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
	}
	return &result, nil
}
