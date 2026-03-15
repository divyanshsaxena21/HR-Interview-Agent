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
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChatService struct {
	interviewCollection   *mongo.Collection
	hrMemoryCollection    *mongo.Collection
	evaluationsCollection *mongo.Collection
	LangchainAgent        *LangChainAgent
}

func NewChatService(interviewCollection, hrMemoryCollection *mongo.Collection) *ChatService {
	return &ChatService{
		interviewCollection: interviewCollection,
		hrMemoryCollection:  hrMemoryCollection,
		LangchainAgent:      NewLangChainAgentWithMemory(hrMemoryCollection),
	}
}

func NewChatServiceWithEvaluations(interviewCollection, hrMemoryCollection, evaluationsCollection *mongo.Collection) *ChatService {
	return &ChatService{
		interviewCollection:   interviewCollection,
		hrMemoryCollection:    hrMemoryCollection,
		evaluationsCollection: evaluationsCollection,
		LangchainAgent:        NewLangChainAgentWithMemory(hrMemoryCollection),
	}
}

func (cs *ChatService) ProcessMessage(ctx context.Context, interviewID primitive.ObjectID, message string) (string, error) {
	log.Printf("[CHAT] ProcessMessage called for interview %s", interviewID.Hex())
	var interview models.Interview
	
	err := cs.interviewCollection.FindOne(ctx, bson.M{"_id": interviewID}).Decode(&interview)
	if err != nil {
		log.Printf("[CHAT] ✗ Error finding interview %s: %v", interviewID.Hex(), err)
		return "", err
	}

	log.Printf("[CHAT] Found interview: stage=%s, hr_questions_asked=%d, availability=%s, rejected=%v", 
		interview.ScreeningStage, interview.HRQuestionsAsked, interview.Availability, interview.Rejected)

	// Check if interview was rejected due to dealbreaker
	if interview.Rejected {
		conclusion := "Thank you for your time. We appreciate your interest, but unfortunately we won't be moving forward at this time. We wish you the best of luck with your career!"
		log.Printf("[CHAT] Interview already rejected, ending with conclusion message")
		return conclusion, nil
	}

	// Initialize stage if not set
	if interview.ScreeningStage == "" {
		interview.ScreeningStage = "questions"
		_, _ = cs.interviewCollection.UpdateOne(ctx, bson.M{"_id": interviewID}, bson.M{
			"$set": bson.M{"screening_stage": "questions"},
		})
	}

	// STAGE 1: Ask Screening Questions
	if interview.ScreeningStage == "questions" {
		// Check if we need to generate a contextual follow-up for the last mandatory question
		if interview.LastQuestionAsked != "" && !interview.FollowUpGenerated && message != "" {
			log.Printf("[CHAT] STAGE 1: Generating contextual follow-up for: %s", interview.LastQuestionAsked)
			followUp, err := cs.GenerateContextualFollowUp(ctx, interview, message)
			if err == nil && followUp != "" {
				// Mark follow-up as generated
				_, _ = cs.interviewCollection.UpdateOne(ctx, bson.M{"_id": interviewID}, bson.M{
					"$set": bson.M{"follow_up_generated": true, "updated_at": time.Now()},
				})
				log.Printf("[CHAT] STAGE 1: Follow-up generated")
				return followUp, nil
			}
			// If follow-up generation fails, continue with next question
		}
		
		// Get next mandatory question
		nextQuestion, err := cs.LangchainAgent.GenerateQuestionWithTracking(interview)
		if err != nil {
			log.Printf("[CHAT] ✗ Error getting next question: %v", err)
			nextQuestion = ""
		}

		// If more questions to ask
		if nextQuestion != "" && nextQuestion != "END_INTERVIEW" {
			log.Printf("[CHAT] STAGE 1: Asking question #%d - %s", interview.HRQuestionsAsked+1, nextQuestion[:min(60, len(nextQuestion))]+"...")
			
			_, err := cs.interviewCollection.UpdateOne(ctx,
				bson.M{"_id": interviewID},
				bson.M{
					"$inc": bson.M{"hr_questions_asked": 1},
					"$set": bson.M{
						"updated_at": time.Now(),
						"last_question_asked": nextQuestion,
						"follow_up_generated": false, // Reset for new question
					},
				},
			)
			if err != nil {
				log.Printf("[CHAT] Error incrementing question counter: %v", err)
			}
			
			return nextQuestion, nil
		}

		// Questions complete, move to next stage
		log.Printf("[CHAT] Questions complete, moving to missing_info stage")
		_, _ = cs.interviewCollection.UpdateOne(ctx, bson.M{"_id": interviewID}, bson.M{
			"$set": bson.M{"screening_stage": "missing_info"},
		})
		interview.ScreeningStage = "missing_info"
	}

	// STAGE 2: Collect Missing Information
	if interview.ScreeningStage == "missing_info" {
		missingInfo := getMissingInfo(interview)
		
		if len(missingInfo) > 0 {
			response := "We're wrapping up the interview. Before we finish, could you please provide the following information:\n"
			for _, info := range missingInfo {
				response += "- " + info + "\n"
			}
			response += "\nYou can share these directly or type 'skip' if they're not available."
			
			log.Printf("[CHAT] STAGE 2: Requesting missing info. Missing: %v", missingInfo)
			return response, nil
		}

		// No missing info or user provided everything, move to availability stage
		log.Printf("[CHAT] Missing info complete, moving to availability stage")
		_, _ = cs.interviewCollection.UpdateOne(ctx, bson.M{"_id": interviewID}, bson.M{
			"$set": bson.M{"screening_stage": "availability"},
		})
		interview.ScreeningStage = "availability"
		
		// After transitioning to availability stage, ask for it without processing current message
		response := "Great! Before we finish, one more thing - could you please share your availability for a meeting with our HR team? You can provide:\n" +
			"- Specific dates and times you're available\n" +
			"- Or your general availability (e.g., 'weekdays after 5 PM', 'flexible')\n\n" +
			"This will help us schedule your next round as quickly as possible."
		
		log.Printf("[CHAT] STAGE 3: Requesting availability")
		return response, nil
	}

	// STAGE 3: Collect HR Meeting Availability
	if interview.ScreeningStage == "availability" {
		// Only process message if we have one (don't process transition message from Stage 2)
		if message != "" && interview.Availability == "" {
			// Validate that the message is actually availability info, not just a generic response
			// Don't save "skip", "ok", "yes", etc. - wait for actual availability
			lowerMsg := strings.ToLower(strings.TrimSpace(message))
			if lowerMsg != "skip" && lowerMsg != "ok" && lowerMsg != "yes" && lowerMsg != "no" && 
			   lowerMsg != "n/a" && lowerMsg != "na" && lowerMsg != "declined" && len(message) > 2 {
				// Save as availability
				if err := cs.SaveAvailability(ctx, interviewID, message); err != nil {
					log.Printf("[CHAT] Error saving availability: %v", err)
				}
				interview.Availability = strings.TrimSpace(message)
				log.Printf("[CHAT] STAGE 3: Availability saved: %s", interview.Availability)
			}
		}
		
		// If still no availability, ask for it
		if interview.Availability == "" {
			response := "Great! Before we finish, one more thing - could you please share your availability for a meeting with our HR team? You can provide:\n" +
				"- Specific dates and times you're available\n" +
				"- Or your general availability (e.g., 'weekdays after 5 PM', 'flexible')\n\n" +
				"This will help us schedule your next round as quickly as possible."
			
			log.Printf("[CHAT] STAGE 3: Requesting availability (no valid response yet)")
			return response, nil
		}

		// Availability collected, move to complete stage
		log.Printf("[CHAT] Availability collected: %s", interview.Availability)
		_, _ = cs.interviewCollection.UpdateOne(ctx, bson.M{"_id": interviewID}, bson.M{
			"$set": bson.M{"screening_stage": "complete"},
		})
		interview.ScreeningStage = "complete"
	}

	// STAGE 4: Interview Complete
	if interview.ScreeningStage == "complete" {
		conclusion := "Thank you for taking the time to interview with us today! We've covered a lot of ground and I appreciate your thoughtful responses. We'll review your interview and get back to you soon with next steps. Have a great day!"
		
		// Mark interview as screening_complete and trigger summary generation
		_, err = cs.interviewCollection.UpdateOne(ctx, bson.M{"_id": interviewID}, bson.M{
			"$set": bson.M{"status": "screening_complete", "updated_at": time.Now()},
		})
		if err != nil {
			log.Printf("[CHAT] Error marking interview as screening_complete: %v", err)
		}

		// Generate screening summary asynchronously
		go func() {
			summaryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			cs.GenerateScreeningSummary(summaryCtx, interviewID)
		}()
		
		log.Printf("[CHAT] STAGE 4: Interview screening complete")
		return conclusion, nil
	}

	// Fallback (shouldn't reach here)
	log.Printf("[CHAT] ✗ Unexpected state: stage=%s", interview.ScreeningStage)
	return "I apologize, there was an issue processing your response. Please try again.", nil
}

func buildProfileInfo(interview models.Interview) string {
	info := ""
	if interview.GitHub != "" {
		info += "\n✓ GitHub: " + interview.GitHub
	}
	if interview.LinkedIn != "" {
		info += "\n✓ LinkedIn: " + interview.LinkedIn
	}
	if interview.Portfolio != "" {
		info += "\n✓ Portfolio: " + interview.Portfolio
	}
	if info == "" {
		info = "\nNone yet - ask for missing links naturally"
	}
	return info
}

func getMissingInfo(interview models.Interview) []string {
	var missing []string
	// Check GitHub (skip if empty, declined, or contains a URL)
	if interview.GitHub == "" || (!strings.Contains(interview.GitHub, "github.com") && interview.GitHub != "declined") {
		if interview.GitHub != "declined" { // Don't ask if explicitly declined
			missing = append(missing, "GitHub profile URL")
		}
	}
	// Check LinkedIn (skip if empty, declined, or contains a URL)
	if interview.LinkedIn == "" || (!strings.Contains(interview.LinkedIn, "linkedin.com") && interview.LinkedIn != "declined") {
		if interview.LinkedIn != "declined" { // Don't ask if explicitly declined
			missing = append(missing, "LinkedIn profile URL")
		}
	}
	// Note: We'll ask for resume/documents separately if needed
	return missing
}

// GetMissingInfoPublic is exported version of getMissingInfo for external use
func GetMissingInfoPublic(interview models.Interview) []string {
	return getMissingInfo(interview)
}

func (cs *ChatService) SaveMessage(ctx context.Context, interviewID primitive.ObjectID, role, content string) error {
	msg := bson.M{
		"role":      role,
		"content":   content,
		"timestamp": time.Now().Unix(),
	}

	_, err := cs.interviewCollection.UpdateOne(ctx,
		bson.M{"_id": interviewID},
		bson.M{
			"$push": bson.M{"messages": msg},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	return err
}

// ExtractAndSaveProfileLinks extracts GitHub, LinkedIn, and Portfolio URLs from candidate message
func (cs *ChatService) ExtractAndSaveProfileLinks(ctx context.Context, interviewID primitive.ObjectID, candidateMessage string) error {
	// Check if candidate is skipping profile links
	lowerMsg := strings.ToLower(strings.TrimSpace(candidateMessage))
	if lowerMsg == "skip" || lowerMsg == "skipped" || lowerMsg == "no" || lowerMsg == "n/a" {
		// Mark all missing fields as "declined" so we don't ask again
		update := bson.M{"$set": bson.M{"updated_at": time.Now()}}
		
		// Fetch current interview to see which fields are missing
		var interview models.Interview
		err := cs.interviewCollection.FindOne(ctx, bson.M{"_id": interviewID}).Decode(&interview)
		if err == nil {
			if interview.GitHub == "" {
				update["$set"].(bson.M)["github"] = "declined"
			}
			if interview.LinkedIn == "" {
				update["$set"].(bson.M)["linkedin"] = "declined"
			}
			if interview.Portfolio == "" {
				update["$set"].(bson.M)["portfolio"] = "declined"
			}
		}
		
		log.Printf("[CHAT] ✓ Candidate skipped profile links")
		err = nil
		_, err = cs.interviewCollection.UpdateOne(ctx, bson.M{"_id": interviewID}, update)
		return err
	}
	
	links := extractProfileLinks(candidateMessage)
	
	if links.GitHub == "" && links.LinkedIn == "" && links.Portfolio == "" {
		return nil // No links to save
	}

	update := bson.M{"$set": bson.M{"updated_at": time.Now()}}
	
	if links.GitHub != "" {
		update["$set"].(bson.M)["github"] = links.GitHub
		log.Printf("[CHAT] ✓ Extracted GitHub: %s", links.GitHub)
	}
	if links.LinkedIn != "" {
		update["$set"].(bson.M)["linkedin"] = links.LinkedIn
		log.Printf("[CHAT] ✓ Extracted LinkedIn: %s", links.LinkedIn)
	}
	if links.Portfolio != "" {
		update["$set"].(bson.M)["portfolio"] = links.Portfolio
		log.Printf("[CHAT] ✓ Extracted Portfolio: %s", links.Portfolio)
	}

	_, err := cs.interviewCollection.UpdateOne(ctx, bson.M{"_id": interviewID}, update)
	return err
}

type ProfileLinks struct {
	GitHub   string
	LinkedIn string
	Portfolio string
}

func extractProfileLinks(message string) ProfileLinks {
	links := ProfileLinks{}
	lowerMsg := strings.ToLower(message)

	// Extract GitHub URL
	githubRegex := regexp.MustCompile(`(https?://)?(?:www\.)?github\.com/[\w\-]+`)
	if matches := githubRegex.FindStringSubmatch(message); len(matches) > 0 {
		links.GitHub = matches[0]
		if !strings.HasPrefix(links.GitHub, "http") {
			links.GitHub = "https://" + links.GitHub
		}
	}

	// Extract LinkedIn URL
	linkedinRegex := regexp.MustCompile(`(https?://)?(?:www\.)?linkedin\.com/(?:in|company)/[\w\-]+`)
	if matches := linkedinRegex.FindStringSubmatch(message); len(matches) > 0 {
		links.LinkedIn = matches[0]
		if !strings.HasPrefix(links.LinkedIn, "http") {
			links.LinkedIn = "https://" + links.LinkedIn
		}
	}

	// Extract Portfolio URL (any other http/https URL that's not GitHub or LinkedIn)
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	if matches := urlRegex.FindAllString(message, -1); len(matches) > 0 {
		for _, url := range matches {
			// Skip GitHub and LinkedIn URLs
			if !strings.Contains(url, "github.com") && !strings.Contains(url, "linkedin.com") {
				links.Portfolio = url
				log.Printf("[CHAT] ✓ Found portfolio URL: %s", url)
				break
			}
		}
	}

	// Also check for common portfolio patterns in text (e.g., "portfolio: domain.com")
	if links.Portfolio == "" && strings.Contains(lowerMsg, "portfolio") {
		// Try to find portfolio domain
		portfolioRegex := regexp.MustCompile(`portfolio[:\s]+([a-zA-Z0-9\.\-]+\.[a-zA-Z]+)`)
		if matches := portfolioRegex.FindStringSubmatch(message); len(matches) > 1 {
			domain := matches[1]
			links.Portfolio = "https://" + domain
		}
	}

	return links
}

func (cs *ChatService) CheckDealbreaker(ctx context.Context, interviewID primitive.ObjectID, message string) (bool, string, error) {
	dealbreakers, err := cs.getDealbreakers(ctx)
	if err != nil {
		return false, "", err
	}

	for _, db := range dealbreakers {
		if matchesDealbreaker(message, db.Question) {
			return true, db.Question, nil
		}
	}
	return false, "", nil
}

func (cs *ChatService) MarkAsRejected(ctx context.Context, interviewID primitive.ObjectID, reason string) error {
	// Fetch the interview to evaluate it
	var interview models.Interview
	err := cs.interviewCollection.FindOne(ctx, bson.M{"_id": interviewID}).Decode(&interview)
	if err != nil {
		log.Printf("[CHAT] Error fetching interview for rejection: %v", err)
		// Continue even if fetch fails - mark as rejected anyway
	} else if cs.evaluationsCollection != nil {
		// Evaluate the interview
		evaluationService := NewEvaluationService()
		evaluation, err := evaluationService.EvaluateInterview(interview)
		if err == nil && evaluation != nil {
			result, err := cs.evaluationsCollection.InsertOne(ctx, evaluation)
			if err == nil {
				log.Printf("[CHAT] ✓ Created evaluation %s for rejected interview %s", result.InsertedID, interviewID.Hex())
				// Update interview with evaluation ID and rejection
				_, _ = cs.interviewCollection.UpdateOne(ctx,
					bson.M{"_id": interviewID},
					bson.M{
						"$set": bson.M{
							"rejected":         true,
							"rejection_reason": reason,
							"status":           "completed",
							"evaluation_id":    result.InsertedID,
							"updated_at":       time.Now(),
						},
					},
				)
				return nil
			}
		}
	}

	// Fallback: mark as rejected without evaluation
	_, err = cs.interviewCollection.UpdateOne(ctx,
		bson.M{"_id": interviewID},
		bson.M{
			"$set": bson.M{
				"rejected":         true,
				"rejection_reason": reason,
				"status":           "completed",
				"updated_at":       time.Now(),
			},
		},
	)
	return err
}

type DealBreakerQuestion struct {
	ID       primitive.ObjectID `bson:"_id"`
	Question string             `bson:"question"`
}

func (cs *ChatService) getDealbreakers(ctx context.Context) ([]DealBreakerQuestion, error) {
	filter := bson.M{"is_dealbreaker": true, "active": true}
	cursor, err := cs.hrMemoryCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var dealbreakers []DealBreakerQuestion
	if err := cursor.All(ctx, &dealbreakers); err != nil {
		return nil, err
	}
	return dealbreakers, nil
}

// TrackAskedQuestions is now a no-op since we use counter-based tracking
func (cs *ChatService) TrackAskedQuestions(ctx context.Context, interviewID primitive.ObjectID) error {
	return nil
}

func matchesDealbreaker(message, question string) bool {
	if len(message) > 0 && (message[0:1] == "n" || message[0:1] == "N") {
		return true
	}
	return false
}

// SaveAvailability stores candidate's availability for HR meeting
func (cs *ChatService) SaveAvailability(ctx context.Context, interviewID primitive.ObjectID, availability string) error {
	_, err := cs.interviewCollection.UpdateOne(ctx,
		bson.M{"_id": interviewID},
		bson.M{
			"$set": bson.M{
				"availability": strings.TrimSpace(availability),
				"updated_at":   time.Now(),
			},
		},
	)
	if err != nil {
		log.Printf("[CHAT] Error saving availability: %v", err)
	} else {
		log.Printf("[CHAT] ✓ Saved candidate availability for interview %s", interviewID.Hex())
	}
	return err
}

// GenerateScreeningSummary creates HR summary for completed screening
func (cs *ChatService) GenerateScreeningSummary(ctx context.Context, interviewID primitive.ObjectID) error {
	log.Printf("[CHAT] Starting screening summary generation for interview %s", interviewID.Hex())

	// Fetch the interview
	var interview models.Interview
	err := cs.interviewCollection.FindOne(ctx, bson.M{"_id": interviewID}).Decode(&interview)
	if err != nil {
		log.Printf("[CHAT] ✗ Error fetching interview for summary: %v", err)
		return err
	}

	// Create screening summary agent
	summaryCollection := cs.interviewCollection.Database().Collection("screening_summaries")
	summaryAgent := NewScreeningSummaryAgent(summaryCollection, cs.interviewCollection)

	// Generate summary
	summary, err := summaryAgent.GenerateScreeningSummary(ctx, interview)
	if err != nil {
		log.Printf("[CHAT] ✗ Error generating summary: %v", err)
		return err
	}

	// Save summary
	err = summaryAgent.SaveScreeningSummary(ctx, summary)
	if err != nil {
		log.Printf("[CHAT] ✗ Error saving summary: %v", err)
		return err
	}

	log.Printf("[CHAT] ✓ Screening summary generated successfully for interview %s", interviewID.Hex())
	return nil
}

// GenerateContextualFollowUp generates a personalized follow-up question based on candidate's response
func (cs *ChatService) GenerateContextualFollowUp(ctx context.Context, interview models.Interview, candidateResponse string) (string, error) {
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	if groqAPIKey == "" {
		log.Printf("[CHAT] GROQ_API_KEY not set, skipping follow-up generation")
		return "", nil
	}

	// Build the prompt for follow-up generation
	systemPrompt := `You are an HR interviewer generating a natural follow-up question. 
You should ask a deeper, more specific follow-up that explores the candidate's answer without repeating the original question.
Keep the follow-up concise (1-2 sentences) and conversational.
Do NOT include any preamble or explanation - just the follow-up question itself.`

	userPrompt := fmt.Sprintf(`Original question: %s

Candidate's response: %s

Role: %s

Generate a natural, specific follow-up question based on their response that explores it deeper.`, 
		interview.LastQuestionAsked, candidateResponse, interview.Role)

	groqRequest := struct {
		Model     string `json:"model"`
		Messages  []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		MaxTokens int `json:"max_tokens"`
	}{
		Model: "llama-3.3-70b-versatile",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens: 200,
	}

	payload, err := json.Marshal(groqRequest)
	if err != nil {
		log.Printf("[CHAT] Error marshaling follow-up request: %v", err)
		return "", err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("[CHAT] Error creating follow-up request: %v", err)
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+groqAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[CHAT] Error calling Groq API for follow-up: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[CHAT] Error reading follow-up response: %v", err)
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[CHAT] Groq API error for follow-up: %s", string(respBody))
		return "", fmt.Errorf("groq api error: %s", string(respBody))
	}

	var groqResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &groqResp); err != nil {
		log.Printf("[CHAT] Error parsing follow-up response: %v", err)
		return "", err
	}

	if len(groqResp.Choices) == 0 {
		log.Printf("[CHAT] No response from Groq for follow-up")
		return "", fmt.Errorf("no response from groq")
	}

	followUp := strings.TrimSpace(groqResp.Choices[0].Message.Content)
	log.Printf("[CHAT] Generated follow-up: %s", followUp[:min(80, len(followUp))])
	return followUp, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
