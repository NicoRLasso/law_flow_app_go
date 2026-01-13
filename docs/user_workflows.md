# User Workflows

This document provides step-by-step guides for common tasks within LawFlowApp.

## 🏁 Getting Started

### 1. Account Creation (Initial Admin)
The first user of a system usually needs to be created via the command line for security.
- Run `make create-user` in the terminal.
- Follow the interactive prompts to set your email and password.
- You will be assigned the `admin` role by default.

### 2. Logging In
- Navigate to the login page (`/login`).
- Enter your registered email and password.
- If successful, you will be redirected to the dashboard.
- If you have not set up a firm yet, you will be prompted to do so.

## 🏢 Setting Up Your Firm

If you are a new administrator and your account is not yet associated with a firm:
- You will be automatically redirected to the **Firm Setup** page.
- Enter your firm's name, country, and contact details.
- Select your preferred timezone and currency.
- Click **Save Firm Settings**.
- Your dashboard will now be initialized for your specific firm.

## 👥 Team Management

### Inviting/Creating New Users
- From the **User Management** section on the dashboard:
- Click **Add New User**.
- Enter the user's name, email, and designated role (Lawyer, Staff, etc.).
- The new user's status will be set to **Active** by default.
- *Note: Only Administrators can create new users.*

## 🔑 Security Tasks

### Recovering a Lost Password
- On the login page, click **Forgot Password?**.
- Enter the email address associated with your account.
- You will receive an email with a secure, one-time-use reset link.
- Click the link in the email to set a new password.
- *Tip: The reset link expires after 1 hour for security.*

## 📈 Daily Operations

### Navigation
- Use the sidebar (or top navigation) to switch between different modules like **Dashboard** and **Users**.
- The dashboard provides a high-level summary of your firm's status and recent activity.

---
*Questions? Contact your system administrator or refer to our [Admin Guide](admin_guide.md).*
