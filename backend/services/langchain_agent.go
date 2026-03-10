package services

import (
	"ai-recruiter/backend/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type LangChainAgent struct {
	memory []models.Message
}

func NewLangChainAgent() *LangChainAgent {
	return &LangChainAgent{
		memory: []models.Message{},
	}
}

func (la *LangChainAgent) GenerateInitialQuestion(role string) (string, error) {
	question := "Tell me about yourself and your background."
	return question, nil
}

func (la *LangChainAgent) GenerateQuestion(transcript []models.Message, role string) (string, error) {
	questionsAsked := 0
	
	for _, msg := range transcript {
		if msg.Role == "ai" {
			questionsAsked++
		}
	}

	groqAPIKey := os.Getenv("GROQ_API_KEY")
	groqAPIURL := os.Getenv("GROQ_API_URL")

	// Determine the last candidate answer and recent context
	lastCandidate := ""
	recent := []string{}
	// gather last 6 messages for context (most recent first)
	for i := len(transcript) - 1; i >= 0 && len(recent) < 6; i-- {
		m := transcript[i]
		recent = append(recent, fmt.Sprintf("%s: %s", m.Role, m.Content))
		if lastCandidate == "" && m.Role == "candidate" {
			lastCandidate = m.Content
		}
	}

	// If GROQ key and URL are set, attempt a network call to generate a follow-up
	if groqAPIKey != "" && groqAPIURL != "" {
		prompt := fmt.Sprintf("You are an HR interviewer. Generate a single concise follow-up interview question for role '%s' that directly follows from the candidate's last answer.\n\nCandidate's last answer: \"%s\"\n\nRecent conversation (most recent first):\n%s\n\nReturn only the single question text. Do not add numbering or commentary.", role, lastCandidate, strings.Join(recent, "\n"))

		payload := map[string]interface{}{
			"prompt":      prompt,
			"max_tokens":  200,
			"temperature": 0.6,
		}
		bodyBytes, _ := json.Marshal(payload)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", groqAPIURL, bytes.NewBuffer(bodyBytes))
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+groqAPIKey)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 12 * time.Second}
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				respBody, _ := ioutil.ReadAll(resp.Body)

				// First, try to parse as JSON and look for common keys
				var parsed map[string]interface{}
				if err := json.Unmarshal(respBody, &parsed); err == nil {
					if q, ok := parsed["question"].(string); ok && q != "" {
						return strings.TrimSpace(q), nil
					}
					if out, ok := parsed["output"].(string); ok && out != "" {
						return strings.TrimSpace(out), nil
					}
					if txt, ok := parsed["text"].(string); ok && txt != "" {
						return strings.TrimSpace(txt), nil
					}
					if choices, ok := parsed["choices"].([]interface{}); ok && len(choices) > 0 {
						if first, ok := choices[0].(map[string]interface{}); ok {
							if t, ok := first["text"].(string); ok && t != "" {
								return strings.TrimSpace(t), nil
							}
						}
						// sometimes choices are strings
						if firstStr, ok := choices[0].(string); ok && firstStr != "" {
							return strings.TrimSpace(firstStr), nil
						}
					}
				}

				// If not JSON parsable into expected keys, treat raw body as text
				raw := strings.TrimSpace(string(respBody))
				if raw != "" {
					return raw, nil
				}
			}
		}
		// If network or parsing error occurred, fall through to local fallback
	}

	// Fallback: craft a context-aware follow-up using the candidate's last answer
	question, err := la.localFollowup(questionsAsked, role, lastCandidate)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(question), nil
}

func (la *LangChainAgent) localFollowup(questionsAsked int, role string, lastCandidate string) (string, error) {
	excerpt := strings.TrimSpace(lastCandidate)
	if excerpt == "" {
		// If we have no candidate content, fall back to generic prompts
		generic := []string{
			"Can you tell me about a challenging project you've worked on?",
			"How do you approach problem-solving in your daily work?",
			"Describe a situation where you had to learn something new quickly.",
			"What's your experience with collaborative development?",
			"Why are you interested in this particular role?",
		}
		return generic[questionsAsked%len(generic)], nil
	}

	// If the candidate answer is very short (name, greeting), ask a broader
	// context-setting question instead of quoting the short phrase.
	words := strings.Fields(excerpt)
	if len(words) < 4 || len(excerpt) < 30 {
		shortTemplates := []string{
			"Thanks — could you briefly summarize your background relevant to this role?",
			"Can you describe your current or most recent role and responsibilities?",
			"What's one project you're most proud of? Please include your role and technologies.",
			"Tell me about your top technical skill and how you apply it at work.",
			"How many years of experience do you have in the primary technologies relevant to this role?",
		}
		return shortTemplates[questionsAsked%len(shortTemplates)], nil
	}

	// Use a short excerpt for the follow-up to keep prompts concise
	if len(excerpt) > 160 {
		excerpt = excerpt[:160]
	}

	// Heuristic follow-up templates that reference the candidate's answer
	templates := []string{
		"You mentioned '%s' — can you give a concrete example and describe the steps you took?",
		"When you said '%s', what was the outcome and what did you learn from it?",
		"Can you expand on '%s' with specific details about your role and the technologies used?",
		"How did you measure success for '%s', and what metrics or signals did you track?",
		"What obstacles did you face with '%s', and how did you overcome them?",
	}

	tpl := templates[questionsAsked%len(templates)]
	return fmt.Sprintf(tpl, excerpt), nil
}

func (la *LangChainAgent) UpdateMemory(role string, content string) {
	la.memory = append(la.memory, models.Message{
		Role:    role,
		Content: content,
	})
}
