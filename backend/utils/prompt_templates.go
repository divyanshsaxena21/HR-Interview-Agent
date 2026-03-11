package utils

import (
	"fmt"
)

// Structured HR interview questions (8 HR-focused questions)
func GetHRInterviewQuestions(role string) []string {
	return []string{
		// Q1: Background
		"Tell me about your background and professional experience.",

		// Q2: Tech/Skills for role
		fmt.Sprintf("What's your experience with the key technologies and skills required for this %s role?", role),

		// Q3: Key Achievement
		"What's a project or accomplishment you're proud of, and what was your role?",

		// Q4: Problem-Solving & Learning
		"Describe a challenging situation you faced at work, how you handled it, and what you learned.",

		// Q5: Teamwork & Collaboration
		"Tell me about a time you worked in a team. What was your role and how did you contribute?",

		// Q6: Motivation & Alignment
		fmt.Sprintf("Why are you interested in this %s position, and how does it align with your career goals?", role),

		// Q7: Availability & Constraints
		"Are you available to start soon? Please mention your notice period or any other constraints.",

		// Q8: Culture & Development
		"What kind of work environment do you thrive in, and what are your expectations from a company in terms of growth and development?",
	}
}

func GetSystemPrompt(role string) string {
	return `You are a professional HR recruiter conducting an initial screening interview.

Interview is structured over 8-10 questions only. After the 10th question, the interview will conclude with a thank you message.

Guidelines:
- Ask one clear question at a time
- Listen carefully to answers
- Evaluate communication skills, technical depth, confidence, and problem-solving ability
- Keep the conversation natural and professional
- Be encouraging and respectful
- Do NOT ask the same question twice
`
}

func GetEvaluationPrompt() string {
	return `Evaluate the interview transcript and provide scores on a scale of 1-10 for:
1. Communication (clarity and articulation)
2. Technical Knowledge (depth and accuracy)
3. Confidence (composure and self-assurance)
4. Problem-Solving (analytical approach and reasoning)

Also identify key strengths and weaknesses.`
}
