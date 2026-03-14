package services

import (
	"ai-recruiter/backend/models"
	"context"
	"log"
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

	log.Printf("[CHAT] Found interview with %d messages, role: %s", len(interview.Messages), interview.Role)

	// Check if interview was rejected due to dealbreaker
	if interview.Rejected {
		conclusion := "Thank you for your time. We appreciate your interest, but unfortunately we won't be moving forward at this time. We wish you the best of luck with your career!"
		log.Printf("[CHAT] Interview already rejected, ending with conclusion message")
		return conclusion, nil
	}

	// Check if interview should end based on message count
	// Now that we're asking both static (8) + memory questions (dealbreakers), allow more messages
	messageCount := len(interview.Messages)
	shouldEndInterview := messageCount >= 16 // ~8 questions with potential dealbreakers

	// Check for missing candidate information
	missingInfo := getMissingInfo(interview)

	// If should end and missing info, ask for it
	if shouldEndInterview && len(missingInfo) > 0 {
		response := "We're wrapping up the interview. Before we finish, could you please provide the following information:\n"
		for _, info := range missingInfo {
			response += "- " + info + "\n"
		}
		response += "\nYou can share these directly or type 'skip' if they're not available."
		
		log.Printf("[CHAT] Requesting missing info. Message count: %d", messageCount)
		return response, nil
	}

	// If should end and no missing info, wrap up interview
	if shouldEndInterview && len(missingInfo) == 0 {
		conclusion := "Thank you for taking the time to interview with us today! We've covered a lot of ground and I appreciate your thoughtful responses. We'll review your interview and get back to you soon. Have a great day!"
		
		// Mark interview as completed and update candidate status to interviewed
		_, err := cs.interviewCollection.UpdateOne(ctx, bson.M{"_id": interviewID}, bson.M{
			"$set": bson.M{"status": "completed", "updated_at": time.Now()},
		})
		if err != nil {
			log.Printf("[CHAT] Error marking interview as completed: %v", err)
		}
		
		// Get candidate ID from interview and update candidate status to interviewed
		var interview map[string]interface{}
		cs.interviewCollection.FindOne(ctx, bson.M{"_id": interviewID}).Decode(&interview)
		if candidateID, ok := interview["candidate_id"].(string); ok {
			candidateObjID, err := primitive.ObjectIDFromHex(candidateID)
			if err == nil {
				cs.interviewCollection.Database().Collection("candidates").UpdateOne(ctx, bson.M{"_id": candidateObjID}, bson.M{
					"$set": bson.M{"status": "interviewed"},
				})
			}
		}
		
		log.Printf("[CHAT] Interview completed. Final message count: %d", messageCount)
		return conclusion, nil
	}

	// During the interview (before reaching 8 messages), ask structured questions
	// Ask mandatory HR questions first, then role-based questions from HR Memory
	nextQuestion, err := cs.LangchainAgent.GenerateQuestionWithTracking(interview)
	if err != nil {
		log.Printf("[CHAT] ✗ Error getting next question: %v", err)
		nextQuestion = ""
	}

	if nextQuestion != "" && nextQuestion != "END_INTERVIEW" {
		log.Printf("[CHAT] ✓ Returning mandatory question at message count %d", messageCount)
		
		// Increment the question counter so next call gets the next question
		_, err := cs.interviewCollection.UpdateOne(ctx,
			bson.M{"_id": interviewID},
			bson.M{
				"$inc": bson.M{"hr_questions_asked": 1},
				"$set": bson.M{"updated_at": time.Now()},
			},
		)
		if err != nil {
			log.Printf("[CHAT] Error incrementing question counter: %v", err)
		}
		
		return nextQuestion, nil
	}

	// Build system prompt for contextual follow-ups (only used if all mandatory questions done)
	systemPrompt := `You are an experienced HR recruiter conducting a screening interview.
Your role is to assess:
- Soft skills (communication, collaboration, problem-solving mindset)
- Cultural fit and motivation
- Work style and team dynamics
- Career growth mindset
- Professionalism and clarity of thought

Guidelines:
1. Listen carefully to what they say and ask meaningful follow-up questions
2. Build rapport and be conversational
3. Focus on WHY they did things, not technical HOW (this is HR screening, not technical interview)
4. Ask about their approach to problems, collaboration style, and interpersonal skills
5. Keep responses concise (2-3 sentences max)
6. Ask ONE follow-up related to what they just shared
7. Show interest in their career growth and aspirations

Example follow-ups based on what they say:
- If they mention a project: "What was the most challenging part of that project, and how did you handle disagreements?"
- If they mention teamwork: "How do you typically handle conflict with team members?"
- If they mention a challenge: "How did your team or manager support you through that?"
- If they mention internship: "What was the most valuable lesson you learned about working in a professional environment?"

Candidate info:` + buildProfileInfo(interview) + `

IMPORTANT: Do NOT ask the mandatory interview questions yourself - those will be asked separately.
Your job is to FOLLOW UP on what they've already answered.`

	// If missing GitHub or LinkedIn, add instruction to ask naturally
	if interview.GitHub == "" || interview.LinkedIn == "" {
		systemPrompt += "\n\nWhen appropriate in this follow-up response, ask for their GitHub or LinkedIn profile."
	}

	log.Printf("[CHAT] Generating contextual follow-up via Groq for interview %s", interviewID.Hex())
	response, err := cs.LangchainAgent.GenerateResponse(systemPrompt, message, interview.Role)
	if err != nil {
		log.Printf("[CHAT] ✗ Error generating response: %v", err)
		return "", err
	}

	log.Printf("[CHAT] ✓ Final response length: %d for interview %s", len(response), interviewID.Hex())
	return response, nil
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
