package services

import (
	"bytes"
	"fmt"
	"html/template"
	"law_flow_app_go/config"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wneessen/go-mail"
)

// Email represents an email message
type Email struct {
	To       []string
	Subject  string
	HTMLBody string
	TextBody string
}

// loadTemplate loads an email template from the templates/emails directory
func loadTemplate(templateName string, data interface{}) (html string, text string, err error) {
	// Get the base path for templates
	basePath := "templates/emails"

	// Load HTML template
	htmlPath := filepath.Join(basePath, templateName+".html")
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read HTML template %s: %v", htmlPath, err)
	}

	htmlTmpl, err := template.New(templateName + ".html").Parse(string(htmlContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML template: %v", err)
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute HTML template: %v", err)
	}

	// Load text template
	textPath := filepath.Join(basePath, templateName+".txt")
	textContent, err := os.ReadFile(textPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read text template %s: %v", textPath, err)
	}

	textTmpl, err := template.New(templateName + ".txt").Parse(string(textContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse text template: %v", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute text template: %v", err)
	}

	return htmlBuf.String(), textBuf.String(), nil
}

// SendEmail sends an email synchronously using SMTP
func SendEmail(cfg *config.Config, email *Email) error {
	// In development mode, log the email instead of sending
	if cfg.Environment == "development" {
		logEmailToConsole(email)
		log.Printf("✅ Email logged successfully (development mode - not actually sent)")
		return nil // Return early in development mode
	}

	// Validate configuration
	if cfg.SMTPUsername == "" || cfg.SMTPPassword == "" {
		return fmt.Errorf("SMTP credentials not configured")
	}

	// Create message
	m := mail.NewMsg()
	if err := m.From(fmt.Sprintf("%s <%s>", cfg.EmailFromName, cfg.EmailFrom)); err != nil {
		return fmt.Errorf("failed to set From address: %w", err)
	}
	if err := m.To(email.To...); err != nil {
		return fmt.Errorf("failed to set To address: %w", err)
	}
	m.Subject(email.Subject)

	// Set body (prefer HTML if available, fallback to text)
	if email.HTMLBody != "" {
		m.SetBodyString(mail.TypeTextHTML, email.HTMLBody)
		if email.TextBody != "" {
			m.SetBodyString(mail.TypeTextPlain, email.TextBody)
		}
	} else if email.TextBody != "" {
		m.SetBodyString(mail.TypeTextPlain, email.TextBody)
	} else {
		return fmt.Errorf("email must have either HTMLBody or TextBody")
	}

	// Parse SMTP port
	port, err := strconv.Atoi(cfg.SMTPPort)
	if err != nil {
		return fmt.Errorf("invalid SMTP port: %v", err)
	}

	// Create client with options
	c, err := mail.NewClient(cfg.SMTPHost,
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(cfg.SMTPUsername),
		mail.WithPassword(cfg.SMTPPassword),
	)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}

	// Send email
	if err := c.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("Email sent successfully to: %v", email.To)
	return nil
}

// logEmailToConsole logs email details to console in development mode
func logEmailToConsole(email *Email) {
	separator := strings.Repeat("=", 80)
	log.Printf("\n%s\n📧 EMAIL (Development Mode - Not Actually Sent)\n%s", separator, separator)
	log.Printf("To: %v", email.To)
	log.Printf("Subject: %s", email.Subject)
	log.Printf("\n--- TEXT BODY ---\n%s", email.TextBody)
	log.Printf("\n--- HTML BODY (first 500 chars) ---\n%s...", truncate(email.HTMLBody, 500))
	log.Printf("%s\n", separator)
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// SendEmailAsync sends an email asynchronously using a goroutine
// This is the recommended method for sending emails in handlers to avoid blocking HTTP responses
func SendEmailAsync(cfg *config.Config, email *Email) {
	// Create a copy of the email to avoid race conditions
	emailCopy := &Email{
		To:       append([]string{}, email.To...),
		Subject:  email.Subject,
		HTMLBody: email.HTMLBody,
		TextBody: email.TextBody,
	}

	// Send in goroutine
	go func(cfg *config.Config, email *Email) {
		if err := SendEmail(cfg, email); err != nil {
			log.Printf("Error sending async email: %v", err)
		}
	}(cfg, emailCopy)
}

// WelcomeEmailData contains data for the welcome email template
type WelcomeEmailData struct {
	UserName string
}

// BuildWelcomeEmail creates a welcome email for new users
func BuildWelcomeEmail(userEmail, userName string) *Email {
	data := WelcomeEmailData{
		UserName: userName,
	}

	htmlBody, textBody, err := loadTemplate("welcome", data)
	if err != nil {
		log.Printf("Error loading welcome email template: %v", err)
		// Fallback to simple text email
		textBody = fmt.Sprintf("Welcome to LawFlow App, %s!", userName)
		htmlBody = fmt.Sprintf("<p>Welcome to LawFlow App, %s!</p>", userName)
	}

	return &Email{
		To:       []string{userEmail},
		Subject:  "Welcome to LawFlow App!",
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
}

// FirmSetupEmailData contains data for the firm setup email template
type FirmSetupEmailData struct {
	UserName string
	FirmName string
}

// BuildFirmSetupEmail creates a confirmation email for firm setup completion
func BuildFirmSetupEmail(userEmail, userName, firmName string) *Email {
	data := FirmSetupEmailData{
		UserName: userName,
		FirmName: firmName,
	}

	htmlBody, textBody, err := loadTemplate("firm_setup", data)
	if err != nil {
		log.Printf("Error loading firm setup email template: %v", err)
		// Fallback to simple text email
		textBody = fmt.Sprintf("Congratulations %s! Your firm %s has been set up successfully.", userName, firmName)
		htmlBody = fmt.Sprintf("<p>Congratulations %s! Your firm %s has been set up successfully.</p>", userName, firmName)
	}

	return &Email{
		To:       []string{userEmail},
		Subject:  "Firm Setup Complete - LawFlow App",
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
}

// PasswordResetEmailData contains data for the password reset email template
type PasswordResetEmailData struct {
	UserName  string
	ResetLink string
	ExpiresAt string
}

// BuildPasswordResetEmail creates a password reset email with reset link
func BuildPasswordResetEmail(userEmail, userName, resetLink, expiresAt string) *Email {
	data := PasswordResetEmailData{
		UserName:  userName,
		ResetLink: resetLink,
		ExpiresAt: expiresAt,
	}

	htmlBody, textBody, err := loadTemplate("password_reset", data)
	if err != nil {
		log.Printf("Error loading password reset email template: %v", err)
		// Fallback to simple text email
		textBody = fmt.Sprintf("Password reset requested for %s. Reset link: %s (expires: %s)", userName, resetLink, expiresAt)
		htmlBody = fmt.Sprintf("<p>Password reset requested for %s.</p><p>Reset link: <a href=\"%s\">%s</a></p><p>Expires: %s</p>", userName, resetLink, resetLink, expiresAt)
	}

	return &Email{
		To:       []string{userEmail},
		Subject:  "Password Reset Request - LawFlow App",
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
}

// CaseRequestRejectionEmailData contains data for the case request rejection email template
type CaseRequestRejectionEmailData struct {
	ClientName    string
	FirmName      string
	RejectionNote string
	FirmEmail     string
	FirmPhone     string
}

// BuildCaseRequestRejectionEmail creates a rejection email for case requests
func BuildCaseRequestRejectionEmail(clientEmail, clientName, firmName, rejectionNote, firmEmail, firmPhone string) *Email {
	data := CaseRequestRejectionEmailData{
		ClientName:    clientName,
		FirmName:      firmName,
		RejectionNote: rejectionNote,
		FirmEmail:     firmEmail,
		FirmPhone:     firmPhone,
	}

	htmlBody, textBody, err := loadTemplate("case_request_rejection", data)
	if err != nil {
		log.Printf("Error loading case request rejection email template: %v", err)
		// Fallback to simple text email
		textBody = fmt.Sprintf("Dear %s,\n\nThank you for your interest in %s. Unfortunately, we are unable to proceed with your case request at this time.\n\nReason:\n%s\n\nIf you have any questions, please contact us at %s or %s.\n\nBest regards,\n%s", clientName, firmName, rejectionNote, firmEmail, firmPhone, firmName)
		htmlBody = fmt.Sprintf("<p>Dear %s,</p><p>Thank you for your interest in %s. Unfortunately, we are unable to proceed with your case request at this time.</p><p><strong>Reason:</strong><br>%s</p><p>If you have any questions, please contact us at %s or %s.</p><p>Best regards,<br>%s</p>", clientName, firmName, rejectionNote, firmEmail, firmPhone, firmName)
	}

	return &Email{
		To:       []string{clientEmail},
		Subject:  fmt.Sprintf("Case Request Update - %s", firmName),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
}

// CaseAcceptanceEmailData contains data for the case acceptance email template
type CaseAcceptanceEmailData struct {
	ClientName string
	FirmName   string
	CaseNumber string
}

// BuildCaseAcceptanceEmail creates a welcome email for clients when their case is accepted
func BuildCaseAcceptanceEmail(clientEmail, clientName, firmName, caseNumber string) *Email {
	data := CaseAcceptanceEmailData{
		ClientName: clientName,
		FirmName:   firmName,
		CaseNumber: caseNumber,
	}

	htmlBody, textBody, err := loadTemplate("case_acceptance", data)
	if err != nil {
		log.Printf("Error loading case acceptance email template: %v", err)
		// Fallback to simple text email
		textBody = fmt.Sprintf("Dear %s,\n\nWe are pleased to inform you that %s has accepted your case request.\n\nCase Number: %s\n\nYour assigned lawyer will contact you shortly.\n\nBest regards,\n%s", clientName, firmName, caseNumber, firmName)
		htmlBody = fmt.Sprintf("<p>Dear %s,</p><p>We are pleased to inform you that %s has accepted your case request.</p><p><strong>Case Number: %s</strong></p><p>Your assigned lawyer will contact you shortly.</p><p>Best regards,<br>%s</p>", clientName, firmName, caseNumber, firmName)
	}

	return &Email{
		To:       []string{clientEmail},
		Subject:  fmt.Sprintf("Case Accepted - %s", firmName),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
}

// LawyerAssignmentEmailData contains data for the lawyer assignment email template
type LawyerAssignmentEmailData struct {
	LawyerName string
	CaseNumber string
	ClientName string
}

// BuildLawyerAssignmentEmail creates an assignment notification email for lawyers
func BuildLawyerAssignmentEmail(lawyerEmail, lawyerName, caseNumber, clientName string) *Email {
	data := LawyerAssignmentEmailData{
		LawyerName: lawyerName,
		CaseNumber: caseNumber,
		ClientName: clientName,
	}

	htmlBody, textBody, err := loadTemplate("lawyer_assignment", data)
	if err != nil {
		log.Printf("Error loading lawyer assignment email template: %v", err)
		// Fallback to simple text email
		textBody = fmt.Sprintf("Dear %s,\n\nA new case has been assigned to you.\n\nCase Number: %s\nClient Name: %s\n\nPlease log in to the dashboard to view the complete case information.\n\nBest regards", lawyerName, caseNumber, clientName)
		htmlBody = fmt.Sprintf("<p>Dear %s,</p><p>A new case has been assigned to you.</p><p><strong>Case Number:</strong> %s<br><strong>Client Name:</strong> %s</p><p>Please log in to the dashboard to view the complete case information.</p>", lawyerName, caseNumber, clientName)
	}

	return &Email{
		To:       []string{lawyerEmail},
		Subject:  fmt.Sprintf("New Case Assigned - %s", caseNumber),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
}
