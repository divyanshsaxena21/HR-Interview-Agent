package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type AssemblyAISTTService struct {
	apiKey string
}

func NewAssemblyAISTTService() *AssemblyAISTTService {
	return &AssemblyAISTTService{
		apiKey: os.Getenv("ASSEMBLYAI_API_KEY"),
	}
}

func (as *AssemblyAISTTService) SpeechToText(audioURL string) (string, error) {
	if as.apiKey == "" {
		return "", fmt.Errorf("ASSEMBLYAI_API_KEY not set")
	}

	// Create a transcript request
	payload := map[string]interface{}{"audio_url": audioURL}
	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("POST", "https://api.assemblyai.com/v2/transcript", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", as.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("assemblyai transcript request failed: %s", string(respBody))
	}

	var respJson map[string]interface{}
	if err := json.Unmarshal(respBody, &respJson); err != nil {
		return "", err
	}

	id, _ := respJson["id"].(string)
	if id == "" {
		return "", fmt.Errorf("no transcript id returned")
	}

	// Poll for completion
	pollURL := fmt.Sprintf("https://api.assemblyai.com/v2/transcript/%s", id)
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		r, err := http.NewRequest("GET", pollURL, nil)
		if err != nil {
			return "", err
		}
		r.Header.Set("Authorization", as.apiKey)

		pr, err := client.Do(r)
		if err != nil {
			return "", err
		}
		bodyBytes, _ := ioutil.ReadAll(pr.Body)
		pr.Body.Close()
		var pj map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &pj); err != nil {
			return "", err
		}
		status, _ := pj["status"].(string)
		if status == "completed" {
			text, _ := pj["text"].(string)
			return text, nil
		}
		if status == "error" {
			return "", fmt.Errorf("transcription error: %v", pj["error"])
		}
	}

	return "", fmt.Errorf("transcription timed out")
}

// UploadBytes uploads raw audio bytes to AssemblyAI and returns an upload_url
func (as *AssemblyAISTTService) UploadBytes(data []byte) (string, error) {
	if as.apiKey == "" {
		return "", fmt.Errorf("ASSEMBLYAI_API_KEY not set")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", "https://api.assemblyai.com/v2/upload", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", as.apiKey)
	req.Header.Set("Transfer-Encoding", "chunked")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("assemblyai upload failed: %s", string(body))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if url, ok := parsed["upload_url"].(string); ok && url != "" {
		return url, nil
	}
	return "", fmt.Errorf("no upload_url in assemblyai response")
}
