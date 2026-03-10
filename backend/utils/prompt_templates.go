package utils

import (
	"fmt"
)

// Structured HR interview questions (8-10 questions total)
func GetHRInterviewQuestions(role string) []string {
	return []string{
		// Q1: Introduction & Background
		"Could you start by introducing yourself and briefly summarizing your professional background relevant to this role?",

		// Q2: Current/Recent Role
		"Can you tell me about your current or most recent role, your key responsibilities, and what you enjoy most about it?",

		// Q3: Relevant Experience
		fmt.Sprintf("What is your experience with the technologies and skills required for this %s position, and can you give an example of how you've applied them?", role),

		// Q4: Project & Achievement
		"Tell me about a specific project you're proud of — what was the goal, your role, and what was the outcome or impact?",

		// Q5: Problem-Solving
		"Can you describe a challenging situation you faced at work, how you approached solving it, and what you learned from the experience?",

		// Q6: Team Collaboration
		"How do you typically work in a team environment? Can you give an example of a time you had to collaborate closely with colleagues?",

		// Q7: Career Motivation
		fmt.Sprintf("What attracted you to this %s role, and how does it align with your career goals?", role),

		// Q8: Current Offers & Relocation
		"Do you have any competing offers at the moment, and are you able to relocate to our job location if required?",

		// Q9: Salary & Expectations (optional, can be used as 9th or 10th)
		"What are your salary expectations for this role, and are there any other benefits or work arrangements that are important to you?",

		// Q10: Questions & Closure
		"Do you have any questions for me about the role, the team, or the company?",
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
