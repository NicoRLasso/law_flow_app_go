package services

import (
	"bytes"
	"fmt"
	"html/template"
	"law_flow_app_go/config"
	"law_flow_app_go/services/i18n"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/resend/resend-go/v2"
)

// buildEmailWithFallback handles the common logic of loading a template or falling back to a plain text string
func buildEmailWithFallback(templateName string, lang string, tmplData interface{}, toEmail string) *Email {
	htmlBody, textBody, err := loadTemplate(templateName, lang, tmplData)
	if err != nil {
		log.Printf("Error loading %s email template for lang %s: %v", templateName, lang, err)
		// Fallback logic handled within specific builders now, or we return empty and let caller handle
		// But better: we return empty strings here and let the specific builder fill in the fallback
		// Actually, let's keep the fallback logic in the caller or pass it in?
		// To follow the previous pattern, we should pass the fallbackHTML/Text.
		// However, with i18n, the fallbacks are also dynamic.
		// Let's change the signature to NOT take fallbacks, but rely on the template.
		// If template fails, we try the default language template.
		// If that fails, we return an empty body which will cause an error in SendEmail?
		// No, for robustnes, let's try default lang.
	}

	// Double check if we failed to load both
	if htmlBody == "" && textBody == "" {
		// Try default language "en"
		if lang != "en" {
			htmlBody, textBody, err = loadTemplate(templateName, "en", tmplData)
			if err != nil {
				log.Printf("Error loading default 'en' template for %s: %v", templateName, err)
			}
		}
	}

	return &Email{
		To: []string{toEmail},
		// Subject is set by caller
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
}

// Email represents an email message
type Email struct {
	To       []string
	Subject  string
	HTMLBody string
	TextBody string
}

// loadTemplate loads an email template from the templates/emails directory
// It attempts to load templateName + "_" + lang + ".html/.txt"
// If not found, it falls back to templateName + ".html/.txt" (which is assumed to be English/Base)
func loadTemplate(templateName string, lang string, data interface{}) (html string, text string, err error) {
	basePath := "templates/emails"

	// Helper to load and execute a single file
	loadAndExec := func(ext string) (string, error) {
		// Try localized first
		path := filepath.Join(basePath, fmt.Sprintf("%s_%s%s", templateName, lang, ext))
		content, err := os.ReadFile(path)
		if err != nil {
			// Fallback to base template
			path = filepath.Join(basePath, templateName+ext)
			content, err = os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("failed to read template %s: %v", path, err)
			}
		}

		tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
		if err != nil {
			return "", fmt.Errorf("failed to parse template %s: %v", path, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("failed to execute template %s: %v", path, err)
		}
		return buf.String(), nil
	}

	htmlContent, err := loadAndExec(".html")
	if err != nil {
		return "", "", err
	}

	textContent, err := loadAndExec(".txt")
	if err != nil {
		return "", "", err
	}

	return htmlContent, textContent, nil
}

// SendEmail sends an email using Resend API
func SendEmail(cfg *config.Config, email *Email) error {
	// In development mode, log the email instead of sending
	if cfg.EmailTestMode {
		logEmailToConsole(email)
		log.Printf("✅ Email logged successfully (development mode - not actually sent)")
		return nil // Return early in development mode
	}

	// Validate configuration
	if cfg.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not configured")
	}

	// Create Resend client
	client := resend.NewClient(cfg.ResendAPIKey)

	// Build the from address
	fromAddress := fmt.Sprintf("%s <%s>", cfg.EmailFromName, cfg.EmailFrom)

	// Create email params
	params := &resend.SendEmailRequest{
		From:    fromAddress,
		To:      email.To,
		Subject: email.Subject,
	}

	// Set body (prefer HTML if available)
	if email.HTMLBody != "" {
		params.Html = email.HTMLBody
	}
	if email.TextBody != "" {
		params.Text = email.TextBody
	}

	// Validate we have at least one body
	if params.Html == "" && params.Text == "" {
		return fmt.Errorf("email must have either HTMLBody or TextBody")
	}

	// Send email via Resend
	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email via Resend: %v", err)
	}

	log.Printf("Email sent successfully via Resend (ID: %s) to: %v", sent.Id, email.To)
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
func BuildWelcomeEmail(userEmail, userName, lang string) *Email {
	data := WelcomeEmailData{
		UserName: userName,
	}

	email := buildEmailWithFallback("welcome", lang, data, userEmail)
	email.Subject = i18n.Translate(lang, "email.subject.welcome")
	return email
}

// FirmSetupEmailData contains data for the firm setup email template
type FirmSetupEmailData struct {
	UserName string
	FirmName string
}

// BuildFirmSetupEmail creates a confirmation email for firm setup completion
func BuildFirmSetupEmail(userEmail, userName, firmName, lang string) *Email {
	data := FirmSetupEmailData{
		UserName: userName,
		FirmName: firmName,
	}

	email := buildEmailWithFallback("firm_setup", lang, data, userEmail)
	email.Subject = i18n.Translate(lang, "email.subject.firm_setup")
	return email
}

// PasswordResetEmailData contains data for the password reset email template
type PasswordResetEmailData struct {
	UserName  string
	ResetLink string
	ExpiresAt string
}

// BuildPasswordResetEmail creates a password reset email with reset link
func BuildPasswordResetEmail(userEmail, userName, resetLink, expiresAt, lang string) *Email {
	data := PasswordResetEmailData{
		UserName:  userName,
		ResetLink: resetLink,
		ExpiresAt: expiresAt,
	}

	email := buildEmailWithFallback("password_reset", lang, data, userEmail)
	email.Subject = i18n.Translate(lang, "email.subject.password_reset")
	return email
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
func BuildCaseRequestRejectionEmail(clientEmail, clientName, firmName, rejectionNote, firmEmail, firmPhone, lang string) *Email {
	data := CaseRequestRejectionEmailData{
		ClientName:    clientName,
		FirmName:      firmName,
		RejectionNote: rejectionNote,
		FirmEmail:     firmEmail,
		FirmPhone:     firmPhone,
	}

	email := buildEmailWithFallback("case_request_rejection", lang, data, clientEmail)
	email.Subject = i18n.Translate(lang, "email.subject.case_request_rejection", map[string]interface{}{"firmName": firmName})
	return email
}

// CaseAcceptanceEmailData contains data for the case acceptance email template
type CaseAcceptanceEmailData struct {
	ClientName string
	FirmName   string
	CaseNumber string
	Password   string
	LoginURL   string
}

// BuildCaseAcceptanceEmail creates a welcome email for clients when their case is accepted
func BuildCaseAcceptanceEmail(clientEmail, clientName, firmName, caseNumber, password, loginURL, lang string) *Email {
	data := CaseAcceptanceEmailData{
		ClientName: clientName,
		FirmName:   firmName,
		CaseNumber: caseNumber,
		Password:   password,
		LoginURL:   loginURL,
	}

	email := buildEmailWithFallback("case_acceptance", lang, data, clientEmail)
	email.Subject = i18n.Translate(lang, "email.subject.case_acceptance", map[string]interface{}{"firmName": firmName})
	return email
}

// LawyerAssignmentEmailData contains data for the lawyer assignment email template
type LawyerAssignmentEmailData struct {
	LawyerName string
	CaseNumber string
	ClientName string
}

// BuildLawyerAssignmentEmail creates an assignment notification email for lawyers
func BuildLawyerAssignmentEmail(lawyerEmail, lawyerName, caseNumber, clientName, lang string) *Email {
	data := LawyerAssignmentEmailData{
		LawyerName: lawyerName,
		CaseNumber: caseNumber,
		ClientName: clientName,
	}

	email := buildEmailWithFallback("lawyer_assignment", lang, data, lawyerEmail)
	email.Subject = i18n.Translate(lang, "email.subject.lawyer_assignment", map[string]interface{}{"caseNumber": caseNumber})
	return email
}

// CollaboratorAddedEmailData contains data for the collaborator added email template
type CollaboratorAddedEmailData struct {
	CollaboratorName string
	CaseNumber       string
	ClientName       string
	AssignedLawyer   string
}

// BuildCollaboratorAddedEmail creates a notification email when a user is added as a case collaborator
func BuildCollaboratorAddedEmail(collaboratorEmail, collaboratorName, caseNumber, clientName, assignedLawyer, lang string) *Email {
	data := CollaboratorAddedEmailData{
		CollaboratorName: collaboratorName,
		CaseNumber:       caseNumber,
		ClientName:       clientName,
		AssignedLawyer:   assignedLawyer,
	}

	email := buildEmailWithFallback("collaborator_added", lang, data, collaboratorEmail)
	email.Subject = i18n.Translate(lang, "email.subject.collaborator_added", map[string]interface{}{"caseNumber": caseNumber})
	return email
}

// AppointmentConfirmationEmailData contains data for appointment confirmation email
type AppointmentConfirmationEmailData struct {
	ClientName      string
	FirmName        string
	Date            string
	Time            string
	Duration        int
	LawyerName      string
	AppointmentType string
	MeetingURL      string
	ManageLink      string
}

// BuildAppointmentConfirmationEmail creates a confirmation email for new appointments
func BuildAppointmentConfirmationEmail(clientEmail string, data AppointmentConfirmationEmailData, lang string) *Email {
	email := buildEmailWithFallback("appointment_confirmation", lang, data, clientEmail)
	email.Subject = i18n.Translate(lang, "email.subject.appointment_confirmation", map[string]interface{}{"firmName": data.FirmName})
	return email
}

// AppointmentReminderEmailData contains data for appointment reminder email
type AppointmentReminderEmailData struct {
	ClientName string
	FirmName   string
	Date       string
	Time       string
	Duration   int
	LawyerName string
	MeetingURL string
	ManageLink string
}

// BuildAppointmentReminderEmail creates a reminder email for upcoming appointments
func BuildAppointmentReminderEmail(clientEmail string, data AppointmentReminderEmailData, lang string) *Email {
	email := buildEmailWithFallback("appointment_reminder", lang, data, clientEmail)
	email.Subject = i18n.Translate(lang, "email.subject.appointment_reminder", map[string]interface{}{"time": data.Time})
	return email
}

// AppointmentCancelledEmailData contains data for appointment cancellation email
type AppointmentCancelledEmailData struct {
	ClientName         string
	FirmName           string
	Date               string
	Time               string
	LawyerName         string
	CancellationReason string
	BookingLink        string
}

// BuildAppointmentCancelledEmail creates a cancellation notification email
func BuildAppointmentCancelledEmail(clientEmail string, data AppointmentCancelledEmailData, lang string) *Email {
	email := buildEmailWithFallback("appointment_cancelled", lang, data, clientEmail)
	email.Subject = i18n.Translate(lang, "email.subject.appointment_cancelled", map[string]interface{}{"firmName": data.FirmName})
	return email
}

// LawyerAppointmentNotificationEmailData contains data for lawyer notification email
type LawyerAppointmentNotificationEmailData struct {
	LawyerName      string
	ClientName      string
	ClientEmail     string
	ClientPhone     string
	Date            string
	Time            string
	Duration        int
	AppointmentType string
	Notes           string
}

// BuildLawyerAppointmentNotificationEmail notifies lawyer of new appointment
func BuildLawyerAppointmentNotificationEmail(lawyerEmail string, data LawyerAppointmentNotificationEmailData, lang string) *Email {
	email := buildEmailWithFallback("lawyer_notification", lang, data, lawyerEmail)
	email.Subject = i18n.Translate(lang, "email.subject.lawyer_appointment_notification", map[string]interface{}{
		"clientName": data.ClientName,
		"date":       data.Date,
		"time":       data.Time,
	})
	return email
}

// NewUserWelcomeEmailData contains data for the new user welcome email
type NewUserWelcomeEmailData struct {
	UserName  string
	UserEmail string
	Password  string
	LoginURL  string
}

// BuildNewUserWelcomeEmail creates a welcome email for new users created by superadmin
func BuildNewUserWelcomeEmail(userEmail, userName, password, loginURL, lang string) *Email {
	data := NewUserWelcomeEmailData{
		UserName:  userName,
		UserEmail: userEmail,
		Password:  password,
		LoginURL:  loginURL,
	}

	email := buildEmailWithFallback("new_user_welcome", lang, data, userEmail)
	email.Subject = i18n.Translate(lang, "email.subject.new_user_welcome")
	return email
}

// SupportTicketNotificationEmailData contains data for support ticket notification
type SupportTicketNotificationEmailData struct {
	AdminName   string
	UserName    string
	UserEmail   string
	TicketID    string
	Subject     string
	MessageBody string
}

// BuildSupportTicketNotificationEmail creates a notification email for superadmins about a new support ticket
func BuildSupportTicketNotificationEmail(adminEmail, adminName, userName, userEmail, ticketID, subject, messageBody, lang string) *Email {
	data := SupportTicketNotificationEmailData{
		AdminName:   adminName,
		UserName:    userName,
		UserEmail:   userEmail,
		TicketID:    ticketID,
		Subject:     subject,
		MessageBody: messageBody,
	}

	// For simplicity, we can use the same subject as the ticket or a prefixed one
	emailSubject := fmt.Sprintf("[%s] %s", i18n.Translate(lang, "email.subject.support_ticket"), subject)

	// Since we haven't created the template yet, let's create a basic fallback text if template fails
	// Or relies on the template existing.
	email := buildEmailWithFallback("support_notification", lang, data, adminEmail)
	email.Subject = emailSubject
	return email
}
