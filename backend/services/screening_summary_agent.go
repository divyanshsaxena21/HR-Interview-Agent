package services

import (
	"ai-recruiter/backend/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ScreeningSummaryAgent struct {
	summaryCollection *mongo.Collection
	interviewCollection *mongo.Collection
}

func NewScreeningSummaryAgent(summaryCollection, interviewCollection *mongo.Collection) *ScreeningSummaryAgent {
	return &ScreeningSummaryAgent{
		summaryCollection: summaryCollection,
		interviewCollection: interviewCollection,
	}
}

// GenerateScreeningSummary analyzes interview and creates HR summary
func (ssa *ScreeningSummaryAgent) GenerateScreeningSummary(ctx context.Context, interview models.Interview) (*models.ScreeningSummary, error) {
	log.Printf("[SUMMARY] Generating screening summary for interview %s, candidate %s", interview.ID.Hex(), interview.CandidateName)

	// Build transcript text for analysis
	transcriptText := ssa.buildTranscriptText(interview.Messages)

	// Get HR questions asked to understand dealbreaker context
	dealBreakerTriggered := interview.Rejected

	// Generate summary via Groq
	summary := ssa.generateSummaryViaGroq(transcriptText, interview.Role, dealBreakerTriggered)
	if summary == nil {
		// Fallback to heuristic if Groq fails
		summary = ssa.heuristicSummary(interview, dealBreakerTriggered)
	}

	// Determine screening status
	screeningStatus := "pass"
	if dealBreakerTriggered {
		screeningStatus = "reject"
	} else if strings.Contains(strings.ToLower(summary.Recommendation), "needs_more_info") {
		screeningStatus = "needs_review"
	}
	summary.ScreeningStatus = screeningStatus

	// Set interview and candidate IDs
	summary.ID = primitive.NewObjectID()
	summary.InterviewID = interview.ID
	summary.CandidateID = interview.CandidateID
	summary.CandidateName = interview.CandidateName
	summary.Role = interview.Role
	summary.CandidateAvailability = interview.Availability
	summary.CreatedAt = time.Now()
	summary.DealBreakerTriggered = dealBreakerTriggered
	summary.DealBreakerReason = interview.RejectionReason

	return summary, nil
}

// SaveScreeningSummary persists the summary to database
func (ssa *ScreeningSummaryAgent) SaveScreeningSummary(ctx context.Context, summary *models.ScreeningSummary) error {
	result, err := ssa.summaryCollection.InsertOne(ctx, summary)
	if err != nil {
		log.Printf("[SUMMARY] ✗ Error saving screening summary: %v", err)
		return err
	}

	// Link summary to interview
	if summary.InterviewID != primitive.NilObjectID {
		_, err = ssa.interviewCollection.UpdateOne(ctx,
			bson.M{"_id": summary.InterviewID},
			bson.M{
				"$set": bson.M{
					"screening_summary_id": result.InsertedID,
					"status": "screening_complete",
					"updated_at": time.Now(),
				},
			},
		)
		if err != nil {
			log.Printf("[SUMMARY] ✗ Error linking summary to interview: %v", err)
		}
	}

	log.Printf("[SUMMARY] ✓ Saved screening summary %s for interview %s", result.InsertedID, summary.InterviewID.Hex())
	return nil
}

// generateSummaryViaGroq uses Groq LLM to analyze interview
func (ssa *ScreeningSummaryAgent) generateSummaryViaGroq(transcript, role string, dealBreakerTriggered bool) *models.ScreeningSummary {
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	if groqAPIKey == "" {
		return nil
	}

	dealBreakerStatus := "No"
	if dealBreakerTriggered {
		dealBreakerStatus = "Yes - Interview automatically rejected"
	}

	prompt := fmt.Sprintf(`You are an HR director analyzing a screening interview transcript.

INTERVIEW TRANSCRIPT:
%s

ROLE APPLIED: %s
DEALBREAKER TRIGGERED: %s

Analyze this screening interview and generate a structured JSON summary with:

1. screening_status: "pass", "reject", or "needs_review"
2. candidate_strengths: list of 2-3 key strengths observed
3. candidate_weaknesses: list of 2-3 areas for improvement
4. missing_information: list of any missing critical information (GitHub, LinkedIn, resume, etc.)
5. recommendation: "schedule_hr_meeting" if should move to HR round, "reject" if should not advance, "needs_more_info" if unclear
6. hr_notes: 2-3 sentences of actionable HR notes for scheduling/next steps

Return ONLY valid JSON without markdown formatting.`, transcript, role, dealBreakerStatus)

	responseText := ssa.callGroqAPI(prompt, groqAPIKey)
	if responseText == "" {
		return nil
	}

	return ssa.parseSummaryJSON(responseText)
}

// callGroqAPI makes request to Groq
func (ssa *ScreeningSummaryAgent) callGroqAPI(prompt, apiKey string) string {
	messages := []map[string]string{
		{
			"role":    "system",
			"content": "You are an expert HR recruiter providing detailed candidate screening summaries.",
		},
		{
			"role":    "user",
			"content": prompt,
		},
	}

	payload := map[string]interface{}{
		"model":      "llama-3.3-70b-versatile",
		"messages":   messages,
		"max_tokens": 1000,
	}

	jsonPayload, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[SUMMARY] ✗ Groq API error: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return ""
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content
	}
	return ""
}

// parseSummaryJSON converts JSON response to ScreeningSummary
func (ssa *ScreeningSummaryAgent) parseSummaryJSON(jsonText string) *models.ScreeningSummary {
	//  Clean up markdown if present
	jsonText = strings.TrimPrefix(jsonText, "```json")
	jsonText = strings.TrimPrefix(jsonText, "```")
	jsonText = strings.TrimSuffix(jsonText, "```")
	jsonText = strings.TrimSpace(jsonText)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		log.Printf("[SUMMARY] ✗ Error parsing JSON response: %v", err)
		return nil
	}

	summary := &models.ScreeningSummary{
		CandidateStrengths:  parseStringArray(parsed, "candidate_strengths"),
		CandidateWeaknesses: parseStringArray(parsed, "candidate_weaknesses"),
		MissingInformation:  parseStringArray(parsed, "missing_information"),
	}

	if v, ok := parsed["screening_status"].(string); ok {
		summary.ScreeningStatus = v
	}
	if v, ok := parsed["recommendation"].(string); ok {
		summary.Recommendation = v
	}
	if v, ok := parsed["hr_notes"].(string); ok {
		summary.HRNotes = v
	}

	return summary
}

// heuristicSummary generates summary without Groq
func (ssa *ScreeningSummaryAgent) heuristicSummary(interview models.Interview, dealBreakerTriggered bool) *models.ScreeningSummary {
	strengths := []string{}
	weaknesses := []string{}

	// Count candidate messages and words
	candidateMsgs := []string{}
	for _, m := range interview.Messages {
		if m.Role == "candidate" {
			candidateMsgs = append(candidateMsgs, m.Content)
		}
	}

	// Basic heuristics
	avgLength := 0
	if len(candidateMsgs) > 0 {
		totalWords := 0
		for _, msg := range candidateMsgs {
			totalWords += len(strings.Fields(msg))
		}
		avgLength = totalWords / len(candidateMsgs)
	}

	if avgLength > 25 {
		strengths = append(strengths, "Clear and detailed communication")
	}
	if strings.Contains(strings.ToLower(interview.GitHub+interview.LinkedIn), "github") || strings.Contains(strings.ToLower(interview.GitHub+interview.LinkedIn), "linkedin") {
		strengths = append(strengths, "Active professional presence")
	}
	if interview.GitHub == "" || interview.LinkedIn == "" {
		weaknesses = append(weaknesses, "Missing GitHub or LinkedIn profile")
	}
	if len(interview.Documents) == 0 {
		weaknesses = append(weaknesses, "No documents uploaded")
	}

	recommendation := "schedule_hr_meeting"
	status := "pass"
	if dealBreakerTriggered {
		recommendation = "reject"
		status = "reject"
	} else if interview.GitHub == "" && interview.LinkedIn == "" {
		recommendation = "needs_more_info"
		status = "needs_review"
	}

	return &models.ScreeningSummary{
		ScreeningStatus:      status,
		CandidateStrengths:   strengths,
		CandidateWeaknesses:  weaknesses,
		Recommendation:       recommendation,
		HRNotes:              "Basic screening completed. " + fmt.Sprintf("Candidate provided %d interview responses.", len(interview.Messages)/2),
	}
}

// buildTranscriptText formats messages for analysis
func (ssa *ScreeningSummaryAgent) buildTranscriptText(messages []models.Message) string {
	var transcript strings.Builder
	for _, msg := range messages {
		role := msg.Role
		if role == "candidate" {
			role = "Candidate"
		} else {
			role = "Recruiter"
		}
		transcript.WriteString(fmt.Sprintf("%s: %s\n\n", role, msg.Content))
	}
	return transcript.String()
}

// Helper function to parse string arrays from JSON
func parseStringArray(data map[string]interface{}, key string) []string {
	if v, ok := data[key].([]interface{}); ok {
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return []string{}
}
