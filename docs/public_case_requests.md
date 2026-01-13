# Public Case Request System

## What It Does

The public case request system allows potential clients to submit case requests to your law firm through a public web form. Each firm gets a unique URL that can be shared with clients, who can then submit their information and upload supporting documents without needing to create an account.

## Key Features

### For Potential Clients
- **Easy Access**: No login required - just visit your firm's unique URL
- **Simple Form**: Fill out personal information, describe the case, and upload a PDF document
- **Colombian ID Support**: Supports all Colombian document types (CC, CE, Pasaporte, NIT, TI)
- **Priority Selection**: Clients can indicate urgency (Low, Medium, High, Urgent)
- **Mobile Friendly**: Works on all devices
- **Confirmation Page**: After submitting, clients see a professional success page with next steps

### For Law Firms
- **Centralized Dashboard**: View all incoming case requests in one place
- **Smart Filtering**: Filter requests by status (Pending, Accepted, Rejected) and priority levels
- **Secure File Access**: Download client-submitted PDFs securely
- **Status Management**: Streamlined accept/reject workflow for quick decision-making
- **Automatic Client Notifications**: Clients receive professional email notifications when their request is rejected
- **Detailed Request View**: Modal interface showing all client information, documents, and case details
- **Role-Based Access**: Only admins and lawyers can view requests
- **Multi-Tenant**: Each firm only sees their own requests

## How to Use It

### Sharing Your Form

Your firm has a unique URL for case requests. The URL is automatically generated from your firm name:

**How it works:**
- Firm name: "Acme Law Firm" → URL: `https://yourdomain.com/firm/acme-law-firm/request`
- Firm name: "García & Associates" → URL: `https://yourdomain.com/firm/garcia-associates/request`
- Firm name: "Rodríguez Legal Services" → URL: `https://yourdomain.com/firm/rodriguez-legal-services/request`

The system automatically:
- Converts spaces to hyphens
- Removes special characters
- Makes everything lowercase
- Ensures uniqueness

Share this URL on:
- Your website
- Social media
- Email signatures
- Business cards
- Marketing materials

### Managing Requests

1. **Access the Dashboard**
   - Login to Law Flow
   - Click "Requests" in the navigation bar

2. **Review Incoming Requests**
   - See all pending requests at a glance
   - Click "Details" button to view complete request information in a modal
   - View client contact information (name, email, phone, document details)
   - Read full case descriptions
   - Download attached documents

3. **Filter and Organize**
   - Filter by status: Pending, Accepted, or Rejected
   - Filter by priority: Urgent, High, Medium, or Low
   - See priority levels at a glance with color-coded badges

4. **Make Decisions**
   - **Accept Requests**: Change status to "Accepted" for cases you'll take on
   - **Reject Requests**: Change status to "Rejected" if you decline the case
     - System prompts you to provide a rejection reason
     - Client automatically receives a professional email with your explanation
     - Email includes your firm's contact information for follow-up questions

### Request Detail View

The detail modal provides a comprehensive view of each case request:

**Personal Information Section**
- Full name and contact details (email, phone)
- Colombian document type and number (CC, CE, Pasaporte, NIT, TI)

**Case Information Section**
- Full case description submitted by the client
- Priority level (color-coded badge)
- Current status with quick status changer

**Documents Section**
- View attached files (if any)
- Download PDFs securely
- See file metadata (name, size)

**Review Information** (when applicable)
- Who reviewed the request
- When the review occurred
- Rejection reason (if rejected)

**Metadata Section**
- Submission date and time
- Client's IP address (for audit purposes)
- User agent information

## Workflow

### Client Submission Process
```
Client fills form → Clicks "Submit Request" → 
Data saved to database → Redirects to success page →
Client sees confirmation with next steps
```

### Success Page Features
After submitting a request, clients are automatically redirected to a confirmation page that shows:
- ✅ Success confirmation with green checkmark
- 📋 What happens next (review timeline, contact method)
- 📞 Firm contact information (phone and email if available)
- 🔄 Option to submit another request

### Firm Review Process
```
Request appears as "Pending" → Lawyer clicks "Details" →
Reviews all information → Decision made →
Status: "Accepted" or "Rejected"
```

### Rejection Workflow
When you reject a case request:
1. Select "Rejected" status in the request detail modal
2. System prompts for rejection reason
3. Enter a professional explanation (e.g., "We specialize in corporate law and your case requires a criminal defense attorney")
4. Click "Confirm Rejection"
5. Client immediately receives an email containing:
   - Professional rejection notification
   - Your explanation/reason
   - Your firm's contact information
   - Encouragement to seek appropriate legal counsel

**Email Benefits:**
- Maintains professional relationships even when declining cases
- Provides transparency to potential clients
- Reduces follow-up inquiries by explaining the decision upfront
- Shows professionalism and respect for the client's time

## Best Practices

### Writing Rejection Messages

When rejecting a case request, your message should be:

**Professional and Respectful**
- Thank the client for considering your firm
- Be courteous even when declining

**Clear and Specific**
- Explain the reason for the decision
- Avoid vague statements

**Helpful When Possible**
- Suggest what type of attorney they should seek
- Provide context about your firm's specialization

**Examples of Good Rejection Messages:**

✅ "Thank you for contacting our firm. We specialize in corporate and business law, and your case requires expertise in family law. We recommend seeking a family law attorney who can better serve your specific needs."

✅ "We appreciate your interest in our services. Unfortunately, we currently have a full caseload and cannot take on additional personal injury cases at this time. We encourage you to contact another qualified attorney in your area soon, as statute of limitations may apply."

✅ "Thank you for reaching out to us. After reviewing your case, we've determined that the matter falls outside our practice areas. We focus primarily on real estate law, and your case would be better served by a criminal defense attorney."

❌ Avoid: "We don't want your case." (Too harsh, unprofessional)
❌ Avoid: "Cannot help." (Too vague, unhelpful)
❌ Avoid: Not providing any reason (Leaves client confused)

### Response Times

- **Pending Requests**: Review within 1-2 business days when possible
- **High Priority/Urgent**: Prioritize these for same-day or next-day review
- **Consistent Communication**: Even rejections are better than no response

## Security & Privacy

- **Firm Isolation**: You only see requests submitted to your firm
- **Secure Uploads**: Only PDF files accepted, maximum 10MB
- **Authenticated Access**: Only admins and lawyers can view requests
- **Audit Trail**: System tracks IP addresses and submission times

## Setup

### First-Time Setup

If you're setting up an existing firm, run the slug migration:
```bash
go run cmd/migrate-slugs/main.go
```

This creates your firm's unique URL identifier.

### Configuration

Add to your `.env` file:
```bash
UPLOAD_DIR=uploads
```

## Who Can Access What

| Role | Public Form | Dashboard | Download Files | Update Status |
|------|-------------|-----------|----------------|---------------|
| Public | ✅ Submit | ❌ | ❌ | ❌ |
| Client | ✅ Submit | ❌ | ❌ | ❌ |
| Staff | ✅ Submit | ❌ | ❌ | ❌ |
| Lawyer | ✅ Submit | ✅ View | ✅ Download | ✅ Update |
| Admin | ✅ Submit | ✅ View | ✅ Download | ✅ Update |

## Common Use Cases

### Marketing Campaign
Share your case request URL in marketing materials. Potential clients can submit requests 24/7, even outside business hours.

### Website Integration
Add a "Request Consultation" button on your website that links to your case request form.

### Email Responses
When someone emails asking about your services, send them your case request URL for a structured submission.

### Social Media
Share your case request URL in social media profiles and posts to make it easy for people to reach out.

## Future Capabilities

Planned enhancements include:
- Email notifications to firm when new requests arrive
- Acceptance confirmation emails to clients
- Automatic case creation from approved requests
- Public tracking codes so clients can check status
- Custom fields specific to your firm's practice areas
- Multi-file upload support
- SMS notifications for urgent requests
