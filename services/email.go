package services

import (
	"bytes"
	"fmt"
	"html/template"
	"law_flow_app_go/config"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/resend/resend-go/v2"
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
func BuildWelcomeEmail(userEmail, userName string) *Email {
	data := WelcomeEmailData{
		UserName: userName,
	}

	htmlBody, textBody, err := loadTemplate("welcome", data)
	if err != nil {
		log.Printf("Error loading welcome email template: %v", err)
		// Fallback to simple text email
		textBody = fmt.Sprintf("Welcome to lexlegalcloud App, %s!", userName)
		htmlBody = fmt.Sprintf("<p>Welcome to lexlegalcloud App, %s!</p>", userName)
	}

	return &Email{
		To:       []string{userEmail},
		Subject:  "Welcome to lexlegalcloud App!",
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
		Subject:  "Firm Setup Complete - lexlegalcloud App",
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
		Subject:  "Password Reset Request - lexlegalcloud App",
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

// CollaboratorAddedEmailData contains data for the collaborator added email template
type CollaboratorAddedEmailData struct {
	CollaboratorName string
	CaseNumber       string
	ClientName       string
	AssignedLawyer   string
}

// BuildCollaboratorAddedEmail creates a notification email when a user is added as a case collaborator
func BuildCollaboratorAddedEmail(collaboratorEmail, collaboratorName, caseNumber, clientName, assignedLawyer string) *Email {
	data := CollaboratorAddedEmailData{
		CollaboratorName: collaboratorName,
		CaseNumber:       caseNumber,
		ClientName:       clientName,
		AssignedLawyer:   assignedLawyer,
	}

	htmlBody, textBody, err := loadTemplate("collaborator_added", data)
	if err != nil {
		log.Printf("Error loading collaborator added email template: %v", err)
		// Fallback to simple text email
		textBody = fmt.Sprintf("Dear %s,\n\nYou have been added as a collaborator on a case.\n\nCase Number: %s\nClient Name: %s\nPrimary Lawyer: %s\n\nPlease log in to the dashboard to view the case details.\n\nBest regards", collaboratorName, caseNumber, clientName, assignedLawyer)
		htmlBody = fmt.Sprintf("<p>Dear %s,</p><p>You have been added as a collaborator on a case.</p><p><strong>Case Number:</strong> %s<br><strong>Client Name:</strong> %s<br><strong>Primary Lawyer:</strong> %s</p><p>Please log in to the dashboard to view the case details.</p>", collaboratorName, caseNumber, clientName, assignedLawyer)
	}

	return &Email{
		To:       []string{collaboratorEmail},
		Subject:  fmt.Sprintf("Added as Collaborator - Case %s", caseNumber),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
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
func BuildAppointmentConfirmationEmail(clientEmail string, data AppointmentConfirmationEmailData) *Email {
	htmlBody, textBody, err := loadTemplate("appointment_confirmation", data)
	if err != nil {
		log.Printf("Error loading appointment confirmation email template: %v", err)
		textBody = fmt.Sprintf("Dear %s,\n\nYour appointment with %s has been confirmed.\n\nDate: %s\nTime: %s\nLawyer: %s\n\nBest regards,\n%s", data.ClientName, data.FirmName, data.Date, data.Time, data.LawyerName, data.FirmName)
		htmlBody = fmt.Sprintf("<p>Dear %s,</p><p>Your appointment with %s has been confirmed.</p><p><strong>Date:</strong> %s<br><strong>Time:</strong> %s<br><strong>Lawyer:</strong> %s</p><p>Best regards,<br>%s</p>", data.ClientName, data.FirmName, data.Date, data.Time, data.LawyerName, data.FirmName)
	}

	return &Email{
		To:       []string{clientEmail},
		Subject:  fmt.Sprintf("Appointment Confirmed - %s", data.FirmName),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
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
func BuildAppointmentReminderEmail(clientEmail string, data AppointmentReminderEmailData) *Email {
	htmlBody, textBody, err := loadTemplate("appointment_reminder", data)
	if err != nil {
		log.Printf("Error loading appointment reminder email template: %v", err)
		textBody = fmt.Sprintf("Dear %s,\n\nReminder: You have an appointment tomorrow with %s.\n\nDate: %s\nTime: %s\nLawyer: %s\n\nBest regards,\n%s", data.ClientName, data.FirmName, data.Date, data.Time, data.LawyerName, data.FirmName)
		htmlBody = fmt.Sprintf("<p>Dear %s,</p><p>Reminder: You have an appointment tomorrow with %s.</p><p><strong>Date:</strong> %s<br><strong>Time:</strong> %s<br><strong>Lawyer:</strong> %s</p><p>Best regards,<br>%s</p>", data.ClientName, data.FirmName, data.Date, data.Time, data.LawyerName, data.FirmName)
	}

	return &Email{
		To:       []string{clientEmail},
		Subject:  fmt.Sprintf("Appointment Reminder - Tomorrow @ %s", data.Time),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
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
func BuildAppointmentCancelledEmail(clientEmail string, data AppointmentCancelledEmailData) *Email {
	htmlBody, textBody, err := loadTemplate("appointment_cancelled", data)
	if err != nil {
		log.Printf("Error loading appointment cancelled email template: %v", err)
		textBody = fmt.Sprintf("Dear %s,\n\nYour appointment with %s has been cancelled.\n\nDate: %s\nTime: %s\n\nTo book a new appointment: %s\n\nBest regards,\n%s", data.ClientName, data.FirmName, data.Date, data.Time, data.BookingLink, data.FirmName)
		htmlBody = fmt.Sprintf("<p>Dear %s,</p><p>Your appointment with %s has been cancelled.</p><p><strong>Date:</strong> %s<br><strong>Time:</strong> %s</p><p><a href=\"%s\">Book a new appointment</a></p><p>Best regards,<br>%s</p>", data.ClientName, data.FirmName, data.Date, data.Time, data.BookingLink, data.FirmName)
	}

	return &Email{
		To:       []string{clientEmail},
		Subject:  fmt.Sprintf("Appointment Cancelled - %s", data.FirmName),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
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
func BuildLawyerAppointmentNotificationEmail(lawyerEmail string, data LawyerAppointmentNotificationEmailData) *Email {
	textBody := fmt.Sprintf("New Appointment Scheduled\n\nDear %s,\n\nA new appointment has been booked:\n\nClient: %s\nEmail: %s\nPhone: %s\nDate: %s\nTime: %s\nDuration: %d minutes\nType: %s\nNotes: %s\n\nPlease log in to view more details.",
		data.LawyerName, data.ClientName, data.ClientEmail, data.ClientPhone, data.Date, data.Time, data.Duration, data.AppointmentType, data.Notes)

	htmlBody := fmt.Sprintf(`<h2>New Appointment Scheduled</h2>
		<p>Dear %s,</p>
		<p>A new appointment has been booked:</p>
		<ul>
			<li><strong>Client:</strong> %s</li>
			<li><strong>Email:</strong> %s</li>
			<li><strong>Phone:</strong> %s</li>
			<li><strong>Date:</strong> %s</li>
			<li><strong>Time:</strong> %s</li>
			<li><strong>Duration:</strong> %d minutes</li>
			<li><strong>Type:</strong> %s</li>
			<li><strong>Notes:</strong> %s</li>
		</ul>
		<p>Please log in to view more details.</p>`,
		data.LawyerName, data.ClientName, data.ClientEmail, data.ClientPhone, data.Date, data.Time, data.Duration, data.AppointmentType, data.Notes)

	return &Email{
		To:       []string{lawyerEmail},
		Subject:  fmt.Sprintf("New Appointment: %s - %s @ %s", data.ClientName, data.Date, data.Time),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
}

// NewUserWelcomeEmailData contains data for the new user welcome email
type NewUserWelcomeEmailData struct {
	UserName  string
	UserEmail string
	Password  string
	LoginURL  string
}

// BuildNewUserWelcomeEmail creates a welcome email for new users created by superadmin
func BuildNewUserWelcomeEmail(userEmail, userName, password, loginURL string) *Email {
	data := NewUserWelcomeEmailData{
		UserName:  userName,
		UserEmail: userEmail,
		Password:  password,
		LoginURL:  loginURL,
	}

	htmlBody, textBody, err := loadTemplate("new_user_welcome", data)
	if err != nil {
		log.Printf("Error loading new user welcome email template: %v", err)
		textBody = fmt.Sprintf("Welcome to lexlegalcloud!\n\nHello %s,\n\nA new account has been created for you.\nUsername: %s\nPassword: %s\n\nPlease log in at: %s", userName, userEmail, password, loginURL)
		htmlBody = fmt.Sprintf("<p>Welcome to lexlegalcloud!</p><p>Hello %s,</p><p>A new account has been created for you.</p><p>Username: %s<br>Password: %s</p><p>Please log in at: <a href=\"%s\">%s</a></p>", userName, userEmail, password, loginURL, loginURL)
	}

	return &Email{
		To:       []string{userEmail},
		Subject:  "Welcome to lexlegalcloud - Your Account Credentials",
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
}
