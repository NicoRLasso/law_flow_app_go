package services

import (
	"context"
	"fmt"
	"law_flow_app_go/services/i18n"
	"time"
)

// BreachNotificationData contains data for breach notification emails
type BreachNotificationData struct {
	ComplianceOfficerName string
	FirmName              string
	AlertType             string
	Description           string
	IPAddress             string
	UserInvolved          string
	Timestamp             string
	ActionRequired        string
}

// BuildBreachNotificationEmail creates a breach notification email for compliance officers
func BuildBreachNotificationEmail(toEmail, officerName, firmName, alertType, description, ip, userID, lang string) *Email {
	ctx := context.WithValue(context.Background(), i18n.LocaleContextKey, lang)

	data := BreachNotificationData{
		ComplianceOfficerName: officerName,
		FirmName:              firmName,
		AlertType:             alertType,
		Description:           description,
		IPAddress:             ip,
		UserInvolved:          userID,
		Timestamp:             time.Now().Format("2006-01-02 15:04:05"),
		ActionRequired:        i18n.T(ctx, "compliance.alerts.action_required"),
	}

	email := buildEmailWithFallback("breach_notification", lang, data, toEmail)
	email.Subject = fmt.Sprintf("[SECURITY ALERT] %s - %s", i18n.T(ctx, "compliance.alerts.breach_detected"), firmName)

	return email
}

// DataDeletionCertificateData contains data for data deletion certificate
type DataDeletionCertificateData struct {
	UserName        string
	UserEmail       string
	FirmName        string
	DeletionDate    string
	CertificateID   string
	DeletedDataList []string
	CompanyAddress  string
}

// BuildDataDeletionCertificateEmail creates a data deletion certificate email
func BuildDataDeletionCertificateEmail(toEmail, userName, firmName, certificateID, lang string, deletedData []string) *Email {
	ctx := context.WithValue(context.Background(), i18n.LocaleContextKey, lang)

	data := DataDeletionCertificateData{
		UserName:        userName,
		UserEmail:       toEmail,
		FirmName:        firmName,
		DeletionDate:    time.Now().Format("2006-01-02"),
		CertificateID:   certificateID,
		DeletedDataList: deletedData,
	}

	email := buildEmailWithFallback("deletion_certificate", lang, data, toEmail)
	email.Subject = fmt.Sprintf("%s - %s", i18n.T(ctx, "compliance.deletion.certificate_title"), firmName)

	return email
}

// ARCORequestConfirmationData contains data for ARCO request confirmation emails
type ARCORequestConfirmationData struct {
	UserName      string
	RequestType   string
	RequestID     string
	SubmittedDate string
	FirmName      string
	Status        string
}

// BuildARCORequestConfirmationEmail creates a confirmation email for ARCO requests
func BuildARCORequestConfirmationEmail(toEmail, userName, requestType, requestID, firmName, lang string) *Email {
	ctx := context.WithValue(context.Background(), i18n.LocaleContextKey, lang)

	data := ARCORequestConfirmationData{
		UserName:      userName,
		RequestType:   i18n.T(ctx, "compliance.arco.type."+requestType),
		RequestID:     requestID,
		SubmittedDate: time.Now().Format("2006-01-02 15:04"),
		FirmName:      firmName,
		Status:        i18n.T(ctx, "compliance.arco.status.PENDING"),
	}

	email := buildEmailWithFallback("arco_confirmation", lang, data, toEmail)
	email.Subject = fmt.Sprintf("%s - %s #%s", i18n.T(ctx, "compliance.arco.title"), i18n.T(ctx, "compliance.arco.type."+requestType), requestID[:8])

	return email
}

// ARCORequestResolvedData contains data for ARCO request resolution emails
type ARCORequestResolvedData struct {
	UserName     string
	RequestType  string
	RequestID    string
	Status       string
	Response     string
	ResolvedDate string
	FirmName     string
}

// BuildARCORequestResolvedEmail creates a resolution notification email for ARCO requests
func BuildARCORequestResolvedEmail(toEmail, userName, requestType, requestID, status, response, firmName, lang string) *Email {
	ctx := context.WithValue(context.Background(), i18n.LocaleContextKey, lang)

	data := ARCORequestResolvedData{
		UserName:     userName,
		RequestType:  i18n.T(ctx, "compliance.arco.type."+requestType),
		RequestID:    requestID,
		Status:       i18n.T(ctx, "compliance.arco.status."+status),
		Response:     response,
		ResolvedDate: time.Now().Format("2006-01-02 15:04"),
		FirmName:     firmName,
	}

	email := buildEmailWithFallback("arco_resolved", lang, data, toEmail)
	email.Subject = fmt.Sprintf("%s %s - #%s", i18n.T(ctx, "compliance.arco.title"), i18n.T(ctx, "compliance.arco.status."+status), requestID[:8])

	return email
}
