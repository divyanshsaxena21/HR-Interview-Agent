package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
)

type EmailService struct {
	// SMTP config
	smtpHost string
	smtpPort int
	smtpUser string
	smtpPass string
	
	// Brevo config
	brevoKey string
	useBrevo bool
	
	// SendGrid config
	sendgridKey string
	useSendGrid bool
	
	// Resend config
	resendKey string
	useResend bool
}

func NewEmailService() *EmailService {
	// Check if Brevo is configured (preferred)
	brevoKey := os.Getenv("BREVO_API_KEY")
	if brevoKey != "" {
		log.Printf("[EMAIL] Using Brevo service")
		return &EmailService{
			brevoKey: brevoKey,
			useBrevo: true,
		}
	}
	
	// Check if Resend is configured
	resendKey := os.Getenv("RESEND_API_KEY")
	if resendKey != "" {
		log.Printf("[EMAIL] Using Resend service")
		return &EmailService{
			resendKey: resendKey,
			useResend: true,
		}
	}
	
	// Check if SendGrid is configured
	sendgridKey := os.Getenv("SENDGRID_API_KEY")
	if sendgridKey != "" {
		log.Printf("[EMAIL] Using SendGrid service")
		return &EmailService{
			sendgridKey: sendgridKey,
			useSendGrid: true,
		}
	}
	
	// Fall back to SMTP
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
	// Check if email service is configured
	if !es.useBrevo && !es.useResend && !es.useSendGrid && es.smtpHost == "" {
		log.Printf("[EMAIL] No email service configured, skipping email to %s", candidateEmail)
		return nil
	}

	// Use frontend URL from environment, default to localhost for dev
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	
	// Remove trailing slash if present
	if len(frontendURL) > 0 && frontendURL[len(frontendURL)-1] == '/' {
		frontendURL = frontendURL[:len(frontendURL)-1]
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
	
	if es.useBrevo {
		return es.sendEmailViaBrevo(candidateEmail, subject, body)
	}
	if es.useResend {
		return es.sendEmailViaResend(candidateEmail, subject, body)
	}
	if es.useSendGrid {
		return es.sendEmailViaSendGrid(candidateEmail, subject, body)
	}
	return es.sendEmail(candidateEmail, subject, body)
}

// sendEmailViaBrevo sends email using Brevo API
func (es *EmailService) sendEmailViaBrevo(to, subject, htmlBody string) error {
	const brevoURL = "https://api.brevo.com/v3/smtp/email"
	
	// Get sender email from env, fallback to default
	senderEmail := os.Getenv("SENDER_EMAIL")
	if senderEmail == "" {
		senderEmail = "noreply@voxhire.ai"
	}
	
	payload := map[string]interface{}{
		"sender": map[string]string{
			"name":  "VoxHire AI",
			"email": senderEmail,
		},
		"to": []map[string]string{
			{
				"email": to,
			},
		},
		"subject": subject,
		"htmlContent": htmlBody,
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[EMAIL] [ERROR] Failed to marshal Brevo payload: %v", err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	client := &http.Client{}
	req, err := http.NewRequest("POST", brevoURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("[EMAIL] [ERROR] Failed to create Brevo request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("api-key", es.brevoKey)
	req.Header.Set("Content-Type", "application/json")
	
	log.Printf("[EMAIL] [DEBUG] Sending email via Brevo to %s with subject: %s", to, subject)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[EMAIL] [ERROR] Brevo request failed: %v", err)
		return fmt.Errorf("brevo request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[EMAIL] [ERROR] Brevo returned status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("brevo error (status %d): %s", resp.StatusCode, string(body))
	}
	
	log.Printf("[EMAIL] [SUCCESS] Email sent via Brevo to %s", to)
	return nil
}

// sendEmailViaResend sends email using Resend API
func (es *EmailService) sendEmailViaResend(to, subject, htmlBody string) error {
	const resendURL = "https://api.resend.com/emails"
	
	// Get sender email from env, fallback to default
	senderEmail := os.Getenv("SENDER_EMAIL")
	if senderEmail == "" {
		// Default: use onboarding@resend.dev which works without domain verification
		senderEmail = "onboarding@resend.dev"
	}
	
	payload := map[string]interface{}{
		"from":    senderEmail,
		"to":      to,
		"subject": subject,
		"html":    htmlBody,
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[EMAIL] [ERROR] Failed to marshal Resend payload: %v", err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	client := &http.Client{}
	req, err := http.NewRequest("POST", resendURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("[EMAIL] [ERROR] Failed to create Resend request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", es.resendKey))
	req.Header.Set("Content-Type", "application/json")
	
	log.Printf("[EMAIL] [DEBUG] Sending email via Resend from %s to %s with subject: %s", senderEmail, to, subject)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[EMAIL] [ERROR] Resend request failed: %v", err)
		return fmt.Errorf("resend request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[EMAIL] [ERROR] Resend returned status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("resend error (status %d): %s", resp.StatusCode, string(body))
	}
	
	log.Printf("[EMAIL] [SUCCESS] Email sent via Resend to %s", to)
	return nil
}

// sendEmailViaSendGrid sends email using SendGrid API
func (es *EmailService) sendEmailViaSendGrid(to, subject, htmlBody string) error {
	const sendgridURL = "https://api.sendgrid.com/v3/mail/send"
	
	// Get sender email from env, fallback to default
	senderEmail := os.Getenv("SENDER_EMAIL")
	if senderEmail == "" {
		senderEmail = "noreply@sendgrid.com"
	}
	
	payload := map[string]interface{}{
		"personalizations": []map[string]interface{}{
			{
				"to": []map[string]string{
					{"email": to},
				},
			},
		},
		"from": map[string]string{
			"email": senderEmail,
			"name":  "VoxHire AI",
		},
		"subject": subject,
		"content": []map[string]string{
			{
				"type":  "text/html",
				"value": htmlBody,
			},
		},
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[EMAIL] [ERROR] Failed to marshal SendGrid payload: %v", err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	client := &http.Client{}
	req, err := http.NewRequest("POST", sendgridURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("[EMAIL] [ERROR] Failed to create SendGrid request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", es.sendgridKey))
	req.Header.Set("Content-Type", "application/json")
	
	log.Printf("[EMAIL] [DEBUG] Sending email via SendGrid to %s with subject: %s", to, subject)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[EMAIL] [ERROR] SendGrid request failed: %v", err)
		return fmt.Errorf("sendgrid request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[EMAIL] [ERROR] SendGrid returned status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("sendgrid error (status %d): %s", resp.StatusCode, string(body))
	}
	
	log.Printf("[EMAIL] [SUCCESS] Email sent via SendGrid to %s", to)
	return nil
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
	if es.smtpHost == "" || es.smtpUser == "" || es.smtpPass == "" {
		log.Printf("[EMAIL] [ERROR] SMTP credentials not configured. SMTP_HOST: %v, SMTP_USER: %v, SMTP_PASS set: %v", 
			es.smtpHost != "", es.smtpUser != "", es.smtpPass != "")
		return fmt.Errorf("SMTP not properly configured")
	}

	addr := fmt.Sprintf("%s:%d", es.smtpHost, es.smtpPort)
	log.Printf("[EMAIL] Connecting to SMTP: %s", addr)

	auth := smtp.PlainAuth("", es.smtpUser, es.smtpPass, es.smtpHost)

	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"utf-8\"\r\n\r\n",
		es.smtpUser, to, subject)

	fullBody := headers + body

	recipients := []string{to}

	log.Printf("[EMAIL] [DEBUG] Attempting to send email from %s to %s with subject: %s", es.smtpUser, to, subject)
	err := smtp.SendMail(addr, auth, es.smtpUser, recipients, []byte(fullBody))
	if err != nil {
		log.Printf("[EMAIL] [ERROR] SMTP SendMail failed: %v", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("[EMAIL] [SUCCESS] Email sent to %s", to)
	return nil
}
