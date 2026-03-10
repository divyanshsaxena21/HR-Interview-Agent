package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type MurfTTSService struct {
	apiKey string
}

func NewMurfTTSService() *MurfTTSService {
	return &MurfTTSService{
		apiKey: os.Getenv("MURF_API_KEY"),
	}
}

func (ms *MurfTTSService) TextToSpeech(text string) (string, error) {
	if ms.apiKey == "" {
		return "", fmt.Errorf("MURF_API_KEY not set")
	}

	apiURL := os.Getenv("MURF_API_URL")
	if apiURL == "" {
		apiURL = "https://api.murf.ai/v1/speech/generate"
	}

	payload := map[string]interface{}{
		"text": text,
	}

	// Determine voice_id: prefer MURF_VOICE env, otherwise query voices and pick first
	voiceEnv := os.Getenv("MURF_VOICE")
	voiceToUse := ""
	if voiceEnv != "" {
		voiceToUse = voiceEnv
	} else {
		// Query Murf voices
		voicesURL := "https://api.murf.ai/v1/speech/voices"
		vcReq, err := http.NewRequest("GET", voicesURL, nil)
		if err == nil {
			vcReq.Header.Set("api-key", ms.apiKey)
			vcReq.Header.Set("token", ms.apiKey)
			vcReq.Header.Set("Authorization", "Bearer "+ms.apiKey)
			vcClient := &http.Client{Timeout: 10 * time.Second}
			vcResp, err := vcClient.Do(vcReq)
			if err == nil {
				defer vcResp.Body.Close()
				vb, _ := ioutil.ReadAll(vcResp.Body)
				// Log voices response for debugging
				log.Printf("Murf voices response: %s", string(vb))
				var anyv interface{}
				if err := json.Unmarshal(vb, &anyv); err == nil {
					switch t := anyv.(type) {
					case []interface{}:
						if len(t) > 0 {
							if first, ok := t[0].(map[string]interface{}); ok {
								if id, ok := first["voiceId"].(string); ok && id != "" {
									voiceToUse = id
								} else if id, ok := first["id"].(string); ok && id != "" {
									voiceToUse = id
								}
							}
						}
					case map[string]interface{}:
						// try voices key
						if listI, ok := t["voices"].([]interface{}); ok && len(listI) > 0 {
							if first, ok := listI[0].(map[string]interface{}); ok {
								if id, ok := first["voiceId"].(string); ok && id != "" {
									voiceToUse = id
								} else if id, ok := first["id"].(string); ok && id != "" {
									voiceToUse = id
								}
							}
						}
						if voiceToUse == "" {
							if arr, ok := t["data"].([]interface{}); ok && len(arr) > 0 {
								if first, ok := arr[0].(map[string]interface{}); ok {
									if id, ok := first["id"].(string); ok && id != "" {
										voiceToUse = id
									}
								}
							}
						}
					}
				}
			}
		}
	}

	if voiceToUse != "" {
		payload["voice_id"] = voiceToUse
	}

	// If still empty, default to a known Murf voice to avoid invalid voice_id errors
	if voiceToUse == "" {
		voiceToUse = "en-US-alina"
		payload["voice_id"] = voiceToUse
	}

	// If we couldn't determine a voice, fall back to a sensible default
	if voiceToUse == "" {
		voiceToUse = "en-US-alina"
		payload["voice_id"] = voiceToUse
	}
	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	// Murf expects `api-key` or `token` header in some setups; set both to be safe
	req.Header.Set("api-key", ms.apiKey)
	req.Header.Set("token", ms.apiKey)
	req.Header.Set("Authorization", "Bearer "+ms.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Log payload for debugging
	log.Printf("Murf TTS payload: %s", string(body))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, _ := ioutil.ReadAll(resp.Body)
	log.Printf("Murf TTS response: %s", string(respBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("murf tts request failed: %s", string(respBytes))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(respBytes, &parsed); err == nil {
		// Try several common fields where audio URL might be returned
		if url, ok := parsed["audioFile"].(string); ok && url != "" {
			return url, nil
		}
		if url, ok := parsed["audio_url"].(string); ok && url != "" {
			return url, nil
		}
		if url, ok := parsed["url"].(string); ok && url != "" {
			return url, nil
		}
		if data, ok := parsed["data"].(map[string]interface{}); ok {
			if url, ok := data["url"].(string); ok && url != "" {
				return url, nil
			}
		}
	}

	// Fallback: return empty but not fatal
	return "", nil
}
