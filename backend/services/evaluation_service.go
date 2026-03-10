package services

import (
	"ai-recruiter/backend/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EvaluationService struct{}

func NewEvaluationService() *EvaluationService {
	return &EvaluationService{}
}

func (es *EvaluationService) EvaluateInterview(interview models.Interview) (*models.Evaluation, error) {
	// If there are no candidate messages, skip evaluation
	candidateCount := 0
	for _, m := range interview.Messages {
		if m.Role == "candidate" && strings.TrimSpace(m.Content) != "" {
			candidateCount++
		}
	}
	if candidateCount == 0 {
		// No candidate responses — return a zeroed evaluation so the dashboard can show 0s
		return &models.Evaluation{
			ID:                  primitive.NewObjectID(),
			InterviewID:         interview.ID,
			CandidateName:       interview.CandidateName,
			Role:                interview.Role,
			CommunicationScore:  0,
			TechnicalScore:      0,
			ConfidenceScore:     0,
			ProblemSolvingScore: 0,
			Strengths:           []string{},
			Weaknesses:          []string{},
			Summary:             "No responses provided",
			Fit:                 "NOT_FIT",
			CreatedAt:           time.Now(),
		}, nil
	}

	// Prefer GROQ when API key+URL are configured; otherwise use local heuristic
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	groqAPIURL := os.Getenv("GROQ_API_URL")

	transcriptText := es.buildTranscriptText(interview.Messages)
	evaluationPrompt := fmt.Sprintf(`
Analyze this interview transcript and provide a detailed evaluation in JSON format:

Transcript:
%s

Return a JSON object with these exact fields:
{
  "communication_score": (1-10),
  "technical_score": (1-10),
  "confidence_score": (1-10),
  "problem_solving_score": (1-10),
  "strengths": ["list", "of", "strengths"],
  "weaknesses": ["list", "of", "weaknesses"],
  "summary": "brief summary",
  "fit": "GOOD_FIT or POSSIBLE_FIT or NOT_FIT"
}
`, transcriptText)

	if groqAPIKey != "" && groqAPIURL != "" {
		evaluationJSON, err := es.callGroqAPI(evaluationPrompt, groqAPIKey)
		if err == nil && strings.TrimSpace(evaluationJSON) != "" {
			eval := es.parseEvaluation(evaluationJSON, interview.ID, interview.CandidateName, interview.Role)
			return eval, nil
		}
		// fall through to heuristic if GROQ fails
	}

	// Heuristic local evaluation
	return es.heuristicEvaluation(interview), nil
}

func (es *EvaluationService) heuristicEvaluation(interview models.Interview) *models.Evaluation {
	// Aggregate candidate messages
	candidateMsgs := []string{}
	totalWords := 0
	for _, m := range interview.Messages {
		if m.Role == "candidate" {
			candidateMsgs = append(candidateMsgs, strings.TrimSpace(m.Content))
			totalWords += len(strings.Fields(m.Content))
		}
	}
	msgCount := len(candidateMsgs)
	avgWords := 0
	if msgCount > 0 {
		avgWords = totalWords / msgCount
	}

	// Technical keywords
	techKeywords := []string{"go", "golang", "python", "java", "javascript", "react", "node", "docker", "kubernetes", "aws", "azure", "sql", "mongodb", "postgres", "typescript", "rust"}
	problemKeywords := []string{"challenge", "problem", "debug", "fix", "issue", "optimiz", "tradeoff", "trouble", "bottleneck"}
	hedgeKeywords := []string{"maybe", "perhaps", "might", "i think", "not sure", "could be", "sort of", "kind of"}

	techHits := 0
	probHits := 0
	hedgeHits := 0
	for _, msg := range candidateMsgs {
		lower := strings.ToLower(msg)
		for _, k := range techKeywords {
			if strings.Contains(lower, k) {
				techHits++
			}
		}
		for _, k := range problemKeywords {
			if strings.Contains(lower, k) {
				probHits++
			}
		}
		for _, k := range hedgeKeywords {
			if strings.Contains(lower, k) {
				hedgeHits++
			}
		}
	}

	// Map heuristics to 1-10 scores
	commScore := 5
	if avgWords >= 60 {
		commScore = 9
	} else if avgWords >= 40 {
		commScore = 8
	} else if avgWords >= 20 {
		commScore = 6
	} else if avgWords >= 10 {
		commScore = 5
	} else if avgWords > 0 {
		commScore = 4
	}

	techScore := 4
	if msgCount > 0 {
		density := float64(techHits) / float64(maxInt(1, totalWords))
		// density per word scaled
		if density > 0.05 {
			techScore = 9
		} else if density > 0.03 {
			techScore = 8
		} else if density > 0.015 {
			techScore = 7
		} else if techHits > 0 {
			techScore = 6
		}
	}

	confidenceScore := 6
	if hedgeHits > 3 {
		confidenceScore = 4
	} else if hedgeHits > 0 {
		confidenceScore = 5
	} else if avgWords > 30 {
		confidenceScore = 8
	}

	problemScore := 5
	if probHits > 2 {
		problemScore = 8
	} else if probHits > 0 {
		problemScore = 7
	}

	strengths := []string{}
	weaknesses := []string{}
	if commScore >= 7 {
		strengths = append(strengths, "Clear communication")
	} else {
		weaknesses = append(weaknesses, "Conciseness or clarity could improve")
	}
	if techScore >= 7 {
		strengths = append(strengths, "Relevant technical knowledge")
	} else {
		weaknesses = append(weaknesses, "Limited technical detail")
	}
	if problemScore >= 7 {
		strengths = append(strengths, "Problem-solving examples")
	}

	fit := "POSSIBLE_FIT"
	if techScore >= 7 && commScore >= 7 {
		fit = "GOOD_FIT"
	} else if techScore < 5 && commScore < 5 {
		fit = "NOT_FIT"
	}

	summary := fmt.Sprintf("Estimated: communication %d, technical %d, confidence %d", commScore, techScore, confidenceScore)

	return &models.Evaluation{
		ID:                  primitive.NewObjectID(),
		InterviewID:         primitive.NewObjectID(),
		CandidateName:       interview.CandidateName,
		Role:                interview.Role,
		CommunicationScore:  commScore,
		TechnicalScore:      techScore,
		ConfidenceScore:     confidenceScore,
		ProblemSolvingScore: problemScore,
		Strengths:           strengths,
		Weaknesses:          weaknesses,
		Summary:             summary,
		Fit:                 fit,
		CreatedAt:           time.Now(),
	}

}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func (es *EvaluationService) buildTranscriptText(transcript []models.Message) string {
	var result strings.Builder
	for _, msg := range transcript {
		result.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}
	return result.String()
}

func (es *EvaluationService) callGroqAPI(prompt string, apiKey string) (string, error) {
	groqURL := os.Getenv("GROQ_API_URL")
	if groqURL == "" {
		// No GROQ endpoint configured; return empty to trigger local heuristic
		return "", fmt.Errorf("GROQ_API_URL not set")
	}

	payload := map[string]interface{}{"prompt": prompt, "max_tokens": 300}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", groqURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("groq API error: %s", string(respBody))
	}

	return string(respBody), nil
}

func (es *EvaluationService) parseEvaluation(jsonStr string, interviewID interface{}, candidateName string, role string) *models.Evaluation {
	var parsed map[string]interface{}
	_ = json.Unmarshal([]byte(jsonStr), &parsed)

	getInt := func(key string, def int) int {
		if v, ok := parsed[key]; ok {
			switch t := v.(type) {
			case float64:
				return int(t)
			case int:
				return t
			case string:
				// try parse
				var iv int
				fmt.Sscanf(t, "%d", &iv)
				if iv > 0 {
					return iv
				}
			}
		}
		return def
	}

	getStringSlice := func(key string) []string {
		out := []string{}
		if v, ok := parsed[key]; ok {
			switch t := v.(type) {
			case []interface{}:
				for _, it := range t {
					if s, ok := it.(string); ok {
						out = append(out, s)
					}
				}
			case string:
				// split by semicolon or comma
				parts := strings.Split(t, ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						out = append(out, p)
					}
				}
			}
		}
		return out
	}

	// Resolve interviewID to ObjectID if possible
	var evaluationID primitive.ObjectID
	if intID, ok := interviewID.(primitive.ObjectID); ok {
		evaluationID = intID
	} else {
		evaluationID = primitive.NewObjectID()
	}

	communication := getInt("communication_score", 5)
	technical := getInt("technical_score", 5)
	confidence := getInt("confidence_score", 5)
	problemSolving := getInt("problem_solving_score", 5)

	strengths := getStringSlice("strengths")
	weaknesses := getStringSlice("weaknesses")
	summary := ""
	fit := "POSSIBLE_FIT"
	if v, ok := parsed["summary"].(string); ok {
		summary = v
	}
	if v, ok := parsed["fit"].(string); ok {
		fit = v
	}

	return &models.Evaluation{
		ID:                  primitive.NewObjectID(),
		InterviewID:         evaluationID,
		CandidateName:       candidateName,
		Role:                role,
		CommunicationScore:  communication,
		TechnicalScore:      technical,
		ConfidenceScore:     confidence,
		ProblemSolvingScore: problemSolving,
		Strengths:           strengths,
		Weaknesses:          weaknesses,
		Summary:             summary,
		Fit:                 fit,
		CreatedAt:           time.Now(),
	}
}
