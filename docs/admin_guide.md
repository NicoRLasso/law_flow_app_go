# Administrator Guide

This guide is intended for users with the **Admin** role. It covers the responsibilities and tools available to manage your legal firm effectively.

## 🔧 Firm Configuration

As an administrator, you are responsible for the core settings of your firm.

- **Profile Management**: Update your firm's name, physical address, and official contact information.
- **Localization**: Ensure the correct timezone is set so that case timelines and session logs are accurate.
- **System Monitoring**: Keep an eye on the collective activity of your team.

## 🛡️ User & Role Management

One of the most critical tasks is managing who has access to your firm's sensitive data.

- **Onboarding**: Create accounts for new lawyers and staff members as they join the firm.
- **Offboarding**: Deactivate users who are no longer with the firm by changing their status to `Inactive`. This prevents them from logging in while preserving their historical activity for audit purposes.
- **Role Assignment**: Ensure users have the minimum necessary permissions. Only assign the `Admin` role to trusted staff who need to manage the firm itself.

## 🧹 Maintenance & Cleanup

While the system handles most maintenance automatically, it's good to be aware of the background tasks:

- **Session Cleanup**: The system automatically clears out old login sessions every hour to keep the database lean and secure.
- **Security Tokens**: Expired password reset tokens are automatically purged.

## 🗝️ Best Practices

1. **Strong Passwords**: Encourage all firm members to use complex passwords and update them regularly.
2. **Review Activity**: Regularly check the user list to ensure only active, authorized employees have access.
3. **Data Accuracy**: Keep the firm's contact information up to date, as this data may be used in official document generation in the future.

---
*For a more detailed look at specific user tasks, see the [User Workflows](user_workflows.md).*
