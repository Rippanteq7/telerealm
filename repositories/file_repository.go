package repositories

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
)

const TelegramBaseURL = "https://mxmxk-tele.hf.space"

type FileRepository interface {
	SendDocument(botToken, chatID string, file io.Reader, fileName string) (string, error)
	GetFileInfo(botToken, fileID string) (string, int, error)
	CheckBotAndChat(botToken, chatID string) (botInfo, chatInfo interface{}, botInChat, botIsAdmin bool, err error)
}

type fileRepository struct{}

func NewFileRepository() FileRepository {
	return &fileRepository{}
}

func (r *fileRepository) SendDocument(botToken, chatID string, file io.Reader, fileName string) (string, error) {
	url := fmt.Sprintf("%s/bot%s/sendDocument", TelegramBaseURL, botToken)

	secureID := uuid.New().String()
	fileExt := filepath.Ext(fileName)
	newFileName := secureID + fileExt

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("chat_id", chatID)

	part, err := writer.CreateFormFile("document", newFileName)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file contents: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %v", err)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}
	log.Printf("[SendDocument] Response from %s: %s", url, string(respBody))

	var sendDocResp struct {
		Ok     bool `json:"ok"`
		Result struct {
			Document struct {
				FileID string `json:"file_id"`
			} `json:"document"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBody, &sendDocResp); err != nil {
		return "", fmt.Errorf("failed to decode JSON response: %v", err)
	}

	if !sendDocResp.Ok {
		return "", fmt.Errorf("telegram API returned not ok status")
	}

	fileID := sendDocResp.Result.Document.FileID

	return fileID, nil
}

func (r *fileRepository) GetFileInfo(botToken, fileID string) (string, int, error) {
	url := fmt.Sprintf("%s/bot%s/getFile?file_id=%s", TelegramBaseURL, botToken, fileID)

	resp, err := http.Get(url)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send GET request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read response body: %v", err)
	}
	log.Printf("[GetFileInfo] Response from %s: %s", url, string(respBody))

	var getFileResp struct {
		Ok     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
			FileSize int    `json:"file_size"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBody, &getFileResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	if !getFileResp.Ok {
		return "", 0, fmt.Errorf("telegram API returned not ok status: %s", resp.Status)
	}

	finalURL := fmt.Sprintf("%s/file/bot%s/%s", TelegramBaseURL, botToken, getFileResp.Result.FilePath)

	return finalURL, getFileResp.Result.FileSize, nil
}

func (r *fileRepository) CheckBotAndChat(botToken, chatID string) (botInfo, chatInfo interface{}, botInChat, botIsAdmin bool, err error) {
	botInfo, err = r.getBotInfo(botToken)
	if err != nil {
		return nil, nil, false, false, fmt.Errorf("failed to get bot info: %v", err)
	}

	chatInfo, err = r.getChatInfo(botToken, chatID)
	if err != nil {
		return nil, nil, false, false, fmt.Errorf("failed to get chat info: %v", err)
	}

	botInChat, botIsAdmin, err = r.checkBotStatus(botToken, chatID, botInfo.(map[string]interface{})["id"].(float64))
	if err != nil {
		return nil, nil, false, false, fmt.Errorf("failed to check bot status: %v", err)
	}

	return botInfo, chatInfo, botInChat, botIsAdmin, nil
}

func (r *fileRepository) getBotInfo(botToken string) (interface{}, error) {
	url := fmt.Sprintf("%s/bot%s/getMe", TelegramBaseURL, botToken)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	log.Printf("[getBotInfo] Response from %s: %s", url, string(respBody))

	var getMeResp struct {
		Ok     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &getMeResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	if !getMeResp.Ok {
		return nil, fmt.Errorf("telegram API returned not ok status: %s", resp.Status)
	}

	return getMeResp.Result, nil
}

func (r *fileRepository) getChatInfo(botToken, chatID string) (interface{}, error) {
	url := fmt.Sprintf("%s/bot%s/getChat", TelegramBaseURL, botToken)

	data := map[string]string{
		"chat_id": chatID,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	log.Printf("[getChatInfo] Response from %s: %s", url, string(respBody))

	var getChatResp struct {
		Ok     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &getChatResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	if !getChatResp.Ok {
		return nil, fmt.Errorf("telegram API returned not ok status: %s", resp.Status)
	}

	return getChatResp.Result, nil
}

func (r *fileRepository) checkBotStatus(botToken, chatID string, botID float64) (bool, bool, error) {
	url := fmt.Sprintf("%s/bot%s/getChatMember", TelegramBaseURL, botToken)

	data := map[string]interface{}{
		"chat_id": chatID,
		"user_id": botID,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, false, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, false, fmt.Errorf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, false, fmt.Errorf("failed to read response body: %v", err)
	}
	log.Printf("[checkBotStatus] Response from %s: %s", url, string(respBody))

	var getChatMemberResp struct {
		Ok     bool `json:"ok"`
		Result struct {
			Status string `json:"status"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &getChatMemberResp); err != nil {
		return false, false, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	if !getChatMemberResp.Ok {
		return false, false, fmt.Errorf("telegram API returned not ok status: %s", resp.Status)
	}

	botInChat := getChatMemberResp.Result.Status != "left" && getChatMemberResp.Result.Status != "kicked"
	botIsAdmin := getChatMemberResp.Result.Status == "administrator" || getChatMemberResp.Result.Status == "creator"

	return botInChat, botIsAdmin, nil
}
