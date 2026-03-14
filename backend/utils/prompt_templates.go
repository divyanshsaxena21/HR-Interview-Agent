package utils

import (
	"fmt"
)

// Structured HR interview questions (8 HR-focused questions)
func GetHRInterviewQuestions(role string) []string {
	return []string{
		// Q1: Background
		"Let's start with the basics - could you tell me a bit about your background and your professional experience so far?",

		// Q2: Tech/Skills for role
		fmt.Sprintf("Since you're interested in the %s position, what relevant technologies or skills do you have experience with? Walk me through a couple that you're most comfortable with.", role),

		// Q3: Key Achievement
		"Can you think of a project or accomplishment that you're really proud of? I'd love to hear about what you did and what your specific role was in making it happen.",

		// Q4: Problem-Solving & Learning
		"Tell me about a time when you faced a challenging situation at work. What happened, how did you tackle it, and what was the key lesson you learned from it?",

		// Q5: Teamwork & Collaboration
		"Most of our work involves collaborating with others. Can you share an example of a time you worked in a team? What did you contribute, and how did you interact with your teammates?",

		// Q6: Motivation & Alignment
		fmt.Sprintf("What is it about this %s role that drew you to apply? How does it fit with where you want to take your career?", role),

		// Q7: Availability & Constraints
		"On a practical note - are you in a position to start soon if we move forward? Is there a notice period or any other constraints we should know about?",

		// Q8: Culture & Development
		"Finally, what kind of work environment helps you do your best work? And what are you looking for from a company in terms of growth and development opportunities?",
	}
}

func GetSystemPrompt(role string) string {
	return `You are a professional HR recruiter conducting an initial screening interview.

Interview is structured over 8-12 questions only. After all questions, the interview will conclude with a thank you message.

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
