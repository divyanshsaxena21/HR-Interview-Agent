package services

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
)

type EmailService struct {
	smtpHost string
	smtpPort int
	smtpUser string
	smtpPass string
}

func NewEmailService() *EmailService {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	return &EmailService{
		smtpHost: os.Getenv("SMTP_HOST"),
		smtpPort: port,
		smtpUser: os.Getenv("SMTP_USER"),
		smtpPass: os.Getenv("SMTP_PASS"),
	}
}

// SendInterviewEmail sends interview invitation to candidate
func (es *EmailService) SendInterviewEmail(candidateEmail, candidateName, sessionID string) error {
	if es.smtpHost == "" {
		log.Printf("[EMAIL] SMTP not configured (SMTP_HOST env var not set), skipping email to %s", candidateEmail)
		return nil
	}

	// Use frontend URL from environment, default to localhost for dev
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	subject := "Your Interview Has Been Scheduled - VoxHire AI"
	interviewURL := fmt.Sprintf("%s/interview/%s", frontendURL, sessionID)

	body := fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
  <h2>Interview Invitation</h2>
  <p>Hello %s,</p>
  <p>Thank you for your interest in our position. Your interview has been scheduled!</p>
  <p><strong>Join your interview here:</strong></p>
  <p>
    <a href="%s" style="display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px;">
      Start Interview
    </a>
  </p>
  <p>Interview Link: <a href="%s">%s</a></p>
  <p>If you have any questions, please reach out to our HR team.</p>
  <p>Best regards,<br>VoxHire AI Team</p>
</body>
</html>
`, candidateName, interviewURL, interviewURL, interviewURL)

	log.Printf("[EMAIL] Sending interview email to %s with URL: %s", candidateEmail, interviewURL)
	return es.sendEmail(candidateEmail, subject, body)
}

// SendRejectionEmail notifies candidate of rejection
func (es *EmailService) SendRejectionEmail(candidateEmail, candidateName string) error {
	if es.smtpHost == "" {
		log.Println("SMTP not configured, skipping email send")
		return nil
	}

	subject := "Application Status Update - VoxHire AI"

	body := fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
  <h2>Application Status</h2>
  <p>Hello %s,</p>
  <p>Thank you for your interest in our position. After careful consideration, we have decided to move forward with other candidates.</p>
  <p>We appreciate your time and encourage you to apply for future positions that match your skills.</p>
  <p>Best regards,<br>VoxHire AI Team</p>
</body>
</html>
`, candidateName)

	return es.sendEmail(candidateEmail, subject, body)
}

// sendEmail sends an email using SMTP
func (es *EmailService) sendEmail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", es.smtpHost, es.smtpPort)

	auth := smtp.PlainAuth("", es.smtpUser, es.smtpPass, es.smtpHost)

	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"utf-8\"\r\n\r\n",
		es.smtpUser, to, subject)

	fullBody := headers + body

	recipients := []string{to}

	err := smtp.SendMail(addr, auth, es.smtpUser, recipients, []byte(fullBody))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent to %s", to)
	return nil
}
