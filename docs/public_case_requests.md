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
- **Smart Filtering**: Filter requests by status (Pending, Reviewing, Converted, Rejected)
- **Secure File Access**: Download client-submitted PDFs securely
- **Status Management**: Track requests through your workflow
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
   - View client contact information
   - Read case descriptions
   - Download attached documents

3. **Filter and Organize**
   - Click filter buttons to view specific statuses
   - See priority levels at a glance with color-coded badges

4. **Update Status**
   - Mark requests as "Reviewing" when you start evaluation
   - Mark as "Converted" when you create a full case
   - Mark as "Rejected" if you decline the case

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
Request appears as "Pending" → Lawyer reviews → 
Status: "Reviewing" → Decision made → 
Status: "Converted" or "Rejected"
```

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
- Email notifications when new requests arrive
- Automatic case creation from approved requests
- Public tracking codes so clients can check status
- Custom fields specific to your firm's practice areas
- Multi-file upload support
