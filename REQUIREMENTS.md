# REQUIREMENTS.md

## 1. Overview
This project is a reusable internal SaaS foundation designed to accelerate the
development of admin dashboards and reporting/analytics tools. It is built for
a small team of 2–5 technical contributors (developers, operations staff, and
admins). The foundation provides authentication, user/role management, audit
logging, theming, and a pre-built UI component library — delivered in a phased
approach using a React + Vite frontend, Go backend, and PostgreSQL database.

## 2. Goals & Success Criteria

| # | Goal | Success Criterion |
|---|------|-------------------|
| 1 | Reusable authentication system | A developer can bootstrap auth (login, JWT, role enforcement) in a new project in < 1 day |
| 2 | Granular access control | Admins can define custom roles with per-feature permissions without code changes |
| 3 | Accelerated dashboard development | A new admin dashboard can be built using foundation components without duplicating UI work |
| 4 | Immutable audit trail | Every state-changing user action is logged with actor, timestamp, and action details |
| 5 | Per-project theming | Each project can configure its own light/dark theme with minimal effort |

## 3. Out of Scope
- Billing and payment processing (no Stripe or subscription management)
- Mobile or native applications (web only)
- Public-facing or marketing pages
- In-app or email notification system
- Multi-tenancy (not planned for v1; see Open Questions)

## 4. Users & Stakeholders

| Role | Description | Volume / Frequency | Technical Level |
|------|-------------|-------------------|-----------------|
| Admin | Manages users, roles, and system settings | Low, as-needed | High |
| Developer | Builds tools on top of the foundation | Daily during development | High |
| Operations / Back-office | Uses tools built on the foundation | Daily | Low–Medium |

## 5. Functional Requirements

### 5.1 Authentication
- **Description**: Users log in with email and password; sessions are managed via stateless JWT access tokens.
- **Actors**: Any registered user
- **Preconditions**: User account exists and is active
- **Steps**:
  1. User navigates to the login page
  2. User enters email and password
  3. System validates credentials
  4. On success, system issues a signed JWT access token
  5. User is redirected to the main dashboard
- **Alternate flows**:
  - Invalid credentials → generic error ("Invalid email or password")
  - Deactivated account → specific error; login denied
  - Expired token → redirect to login
- **Postconditions**: JWT stored client-side (httpOnly cookie); user is authenticated

### 5.2 User Management
- **Description**: Admins create, view, edit, and deactivate user accounts and assign roles.
- **Actors**: Admin
- **Preconditions**: Actor is authenticated with admin-level permissions
- **Steps**:
  1. Admin opens the User Management screen
  2. Admin views paginated, searchable user list
  3. Admin creates a new user (name, email, role, temporary password)
  4. Admin edits an existing user's profile or role assignment
  5. Admin deactivates a user (soft-delete — data preserved, login blocked)
- **Alternate flows**:
  - Deleting the only admin → system blocks deletion with an error
  - Duplicate email on creation → validation error returned
- **Postconditions**: Changes persisted in DB; action recorded in audit log

### 5.3 Custom Role & Permission Management
- **Description**: Admins define roles with granular per-feature, per-action permissions (read / write / delete).
- **Actors**: Admin
- **Preconditions**: Actor is authenticated with admin-level permissions
- **Steps**:
  1. Admin opens the Roles screen
  2. Admin creates a role with a name and description
  3. Admin assigns per-feature permissions (read / write / delete)
  4. Admin saves the role
  5. Admin assigns the role to users
- **Alternate flows**:
  - Editing an in-use role → changes apply immediately to all assigned users
  - Deleting a role in use → system warns and requires user reassignment before deletion
- **Postconditions**: Role persisted in DB; affected users' permissions update immediately

### 5.4 Audit Logs
- **Description**: The system automatically records all successful state-changing actions.
- **Actors**: System (automatic write); Admin (read/export)
- **Preconditions**: User must be authenticated for action to be logged
- **Steps**:
  1. User performs a state-changing action
  2. System records: actor ID, action type, affected resource, timestamp, before/after values
  3. Admin opens the Audit Log screen
  4. Admin filters by date range, user, or action type
  5. Admin exports filtered results as CSV
- **Postconditions**: Log entry persisted; immutable (no edit or delete of log entries)

### 5.5 UI Component Library
- **Description**: Pre-built React components (built on shadcn/ui) providing consistent UI across all tools.
- **Components**:
  - **Data Table**: Sortable, filterable, paginated; configurable columns
  - **Charts**: Line, bar, and pie charts (via Recharts or equivalent)
  - **Forms**: Controlled inputs with validation, error states, and helper text
  - **Navigation**: Collapsible sidebar, top navigation bar, breadcrumbs

### 5.6 Dark Mode / Theming
- **Description**: Users can toggle light/dark mode; developers can configure per-project theme tokens.
- **Steps**:
  1. User clicks the theme toggle
  2. UI switches color scheme instantly (no page reload)
  3. Preference is persisted (localStorage or user profile)
- **Postconditions**: Theme applied globally; persists across sessions

## 6. Non-Functional Requirements

| Category | Requirement | Priority |
|----------|-------------|----------|
| Performance | API response time < 500ms for standard CRUD under normal load | Medium |
| Availability | No formal SLA — internal tooling, best-effort | Low |
| Security | All endpoints protected by JWT; passwords hashed with bcrypt or argon2 | High |
| Security | JWT signed with HS256 or RS256; access tokens expire ≤ 1 hour | High |
| Security | RBAC enforced server-side on every API endpoint | High |
| Scalability | Designed for < 10 concurrent users; horizontal scaling not required for v1 | Low |
| Data Retention | Audit logs retained indefinitely in v1 (no auto-purge); to be revisited | Low |
| Accessibility | WCAG 2.1 AA is a nice-to-have; not a hard requirement for v1 | Low |

## 7. Integrations & External Dependencies

| System | Purpose | Data Exchanged | Owner |
|--------|---------|---------------|-------|
| shadcn/ui | UI component primitives | Component definitions | Open source |
| Recharts | Chart rendering | Data arrays | Open source |
| PostgreSQL | Primary data store | All application data | Internal |

No external SaaS integrations required for v1.

## 8. Constraints
- **Frontend**: React + Vite
- **Backend**: Go
- **Database**: PostgreSQL
- **API style**: RESTful HTTP
- **No mobile**: Web browser only
- **Hosting**: Undecided — architecture must not be coupled to a specific cloud provider
- **Team**: 2–5 developers; codebase must be maintainable by the full team

## 9. Assumptions

1. **Single-tenant**: Foundation supports one organization for v1. If multi-tenancy is needed, the data schema will require significant changes.
2. **No email service in v1**: No password reset or email verification flow requiring an SMTP/email provider. If needed, an email integration must be added.
3. **JWT stored in httpOnly cookie**: Standard secure storage. If regulated-data compliance arises, server-side sessions may be required instead.
4. **shadcn/ui as UI base**: Chosen due to no stated preference. If a different design system is mandated, significant UI rework will be needed.
5. **Audit logs cover successful actions only**: Failed/denied actions are not logged. If compliance requirements emerge, this must be revisited.

## 10. MVP Definition
Phase 1 (MVP) delivers:
- Email/password authentication with JWT session management
- User management UI (create, edit, deactivate)
- Custom role and permission management (per-feature, per-action)

This is sufficient to bootstrap a secure, role-protected internal tool with full user administration.

## 11. Phasing

| Phase | Deliverables | Dependencies |
|-------|-------------|--------------|
| Phase 1 (MVP) | Authentication, User Management UI, Custom Role Builder | — |
| Phase 2 | UI Component Library (tables, charts, forms, nav), Audit Logs, Dark Mode / Theming | Phase 1 complete |

## 12. Open Questions & Risks

| # | Question / Risk | Owner | Status |
|---|----------------|-------|--------|
| 1 | Is single-tenant scope confirmed? Should the data model support multi-tenancy from day one? | Stakeholder | Open |
| 2 | Is a "forgot password" flow needed for v1? Requires an email service integration. | Stakeholder | Open |
| 3 | Where will this be deployed? Affects environment config and secret management. | Stakeholder | Open |
| 4 | Should failed/denied actions also be captured in audit logs? | Stakeholder | Open |
| 5 | Should JWT include refresh tokens, or is re-login on token expiry acceptable? | Technical | Open |
