# SessionDB Documentation

Welcome to the SessionDB documentation. This directory contains comprehensive technical documentation for the SessionDB project.

## 📚 Documentation Index

### 1. [Backend Requirements Document](./backend-requirements.md)
**Purpose**: Comprehensive backend system requirements and architecture

**Contents**:
- System architecture and technology stack
- Data models and database schema
- Authentication and authorization mechanisms
- Functional requirements for all features
- Non-functional requirements (performance, security, scalability)
- Integration requirements
- Deployment and infrastructure specifications

**Target Audience**: Backend developers, system architects, DevOps engineers

---

### 2. [API Documentation](./api-documentation.md)
**Purpose**: Complete REST API reference for SessionDB

**Contents**:
- Authentication APIs (login, SSO, token refresh)
- User Management APIs (CRUD operations)
- Role Management APIs (role templates and permissions)
- Query Execution APIs (SQL execution, validation, history)
- Approval Workflow APIs (request, approve, reject)
- Audit Log APIs (logging, filtering, export)
- Schema APIs (database discovery, table metadata)
- System APIs (health checks, configuration)
- Error handling and rate limiting
- Webhook integration

**Target Audience**: Frontend developers, API consumers, integration partners

---

## 🚀 Quick Start

### For Backend Developers
1. Read the [Backend Requirements Document](./backend-requirements.md) to understand:
   - System architecture
   - Data models
   - Security requirements
   - Business logic

2. Review the [API Documentation](./api-documentation.md) to understand:
   - API endpoints and contracts
   - Request/response formats
   - Authentication flow

### For Frontend Developers
1. Start with the [API Documentation](./api-documentation.md)
2. Focus on the authentication flow
3. Review the data models in the [Backend Requirements](./backend-requirements.md)

### For Product Managers
1. Review the functional requirements in [Backend Requirements](./backend-requirements.md)
2. Understand the user workflows and approval processes

---

## 🏗️ Project Overview

**SessionDB** is an enterprise-grade database management and access control system that provides:

### Core Features
- ✅ **Secure Database Access**: Role-based access control with granular permissions
- ✅ **Session-Based Users**: Temporary user accounts with auto-expiry
- ✅ **SQL Query Execution**: Safe query execution with permission validation
- ✅ **Approval Workflows**: Multi-step approval for privilege escalation
- ✅ **Comprehensive Auditing**: Full audit trail for compliance
- ✅ **Schema Discovery**: Automatic database schema synchronization
- ✅ **Multi-Database Support**: Connect to multiple databases (PostgreSQL, MySQL, SQL Server)

### Key Benefits
- **Security First**: Deny-by-default permissions, audit logging, SQL injection prevention
- **Compliance Ready**: SOC 2, GDPR, HIPAA-compliant audit trails
- **Developer Friendly**: RESTful APIs, comprehensive documentation
- **Enterprise Scale**: Horizontal scaling, high availability, 99.9% uptime

---

## 📋 Feature Breakdown

### User Management
- Create and manage users with role assignment
- Session-based temporary users with auto-expiry
- User-specific permission grants
- SSO integration (SAML, OAuth, OIDC)

### Role Management
- Define role templates with baseline permissions
- Granular database and table-level permissions
- Permission types: READ, WRITE, DELETE, EXECUTE, ALL
- Wildcard support for databases and tables

### Query Execution
- SQL query execution with permission validation
- Query syntax validation
- Query history tracking
- Saved query scripts
- Multi-tab query editor
- Result pagination and export

### Approval Workflows
- Request temporary access
- Request privilege escalation
- Partial approval support
- Auto-rejection on expiry
- Email/Slack notifications

### Audit & Compliance
- Comprehensive activity logging
- Query execution logging
- Access control change tracking
- Advanced filtering and search
- CSV/PDF export for compliance reports

---

## 🔐 Security Features

### Authentication
- JWT-based authentication
- SSO integration (Okta, Azure AD, Google Workspace)
- Password complexity requirements
- 90-day password rotation
- Multi-factor authentication (planned)

### Authorization
- Role-Based Access Control (RBAC)
- Database and table-level permissions
- Temporary permission grants
- Permission expiry handling
- Deny-by-default security model

### Data Protection
- TLS 1.3 encryption in transit
- AES-256 encryption at rest
- Parameterized queries (SQL injection prevention)
- Rate limiting (100 req/min per user)
- Session security (HttpOnly, Secure cookies)

---

## 🛠️ Technology Stack

### Backend (Recommended)
- **Framework**: Node.js + Express.js OR Python + FastAPI
- **Database**: PostgreSQL (metadata), Multi-database support (target DBs)
- **Cache**: Redis (sessions, schema cache)
- **Queue**: RabbitMQ/Redis (async workflows)
- **Authentication**: JWT, SAML, OAuth 2.0

### Frontend (Current)
- **Framework**: React 18 + TypeScript
- **Routing**: React Router v6
- **State Management**: React Context API
- **Styling**: CSS Modules
- **Icons**: Lucide React
- **Build Tool**: Vite

### Infrastructure
- **Containerization**: Docker
- **Orchestration**: Kubernetes
- **CI/CD**: GitHub Actions / GitLab CI
- **Monitoring**: Prometheus + Grafana
- **Logging**: ELK Stack / Splunk

---

## 📊 Data Models

### Core Entities
- **User**: User accounts with role assignment
- **Role**: Role templates with permissions
- **Permission**: Database/table-level access grants
- **ApprovalRequest**: Access request workflows
- **AuditLog**: Activity and compliance logging
- **QueryHistory**: Query execution tracking
- **SavedScript**: Reusable query templates
- **DatabaseSchema**: Cached schema metadata

For detailed schema definitions, see [Backend Requirements - Data Models](./backend-requirements.md#3-data-models)

---

## 🔄 API Overview

### Base URL
```
Production:  https://api.sessiondb.com/v1
Staging:     https://api-staging.sessiondb.com/v1
Development: http://localhost:3000/v1
```

### Authentication
All API requests require a JWT token:
```
Authorization: Bearer <access_token>
```

### Key Endpoints

#### Authentication
- `POST /auth/login` - User login
- `POST /auth/sso/initiate` - SSO login
- `POST /auth/refresh` - Refresh token
- `POST /auth/logout` - Logout

#### Users
- `GET /users` - List users
- `POST /users` - Create user
- `PUT /users/:id` - Update user
- `DELETE /users/:id` - Deactivate user

#### Roles
- `GET /roles` - List roles
- `POST /roles` - Create role
- `PUT /roles/:id` - Update role
- `DELETE /roles/:id` - Delete role

#### Query Execution
- `POST /query/execute` - Execute SQL query
- `POST /query/validate` - Validate SQL syntax
- `GET /query/history` - Query history
- `GET /query/scripts` - Saved scripts

#### Approvals
- `GET /approvals` - List requests
- `POST /approvals` - Create request
- `POST /approvals/:id/approve` - Approve request
- `POST /approvals/:id/reject` - Reject request

#### Audit Logs
- `GET /audit-logs` - Query audit logs
- `POST /audit-logs/export` - Export logs

For complete API reference, see [API Documentation](./api-documentation.md)

---

## 🚦 Getting Started with Development

### Prerequisites
- Node.js 18+ (for frontend)
- PostgreSQL 14+ (for backend metadata)
- Redis 7+ (for caching)
- Docker & Docker Compose (optional)

### Frontend Setup
```bash
cd /Users/mouli.b/Documents/personal/projects/sessiondb
npm install
npm run dev
```

### Backend Setup (To Be Implemented)
```bash
# Create backend directory
mkdir backend && cd backend

# Initialize Node.js project
npm init -y
npm install express pg redis jsonwebtoken bcrypt

# OR Initialize Python project
python -m venv venv
source venv/bin/activate
pip install fastapi uvicorn sqlalchemy redis pyjwt bcrypt
```

---

## 📈 Roadmap

### Phase 1: Core Backend (Current)
- [ ] Implement authentication APIs
- [ ] Implement user management APIs
- [ ] Implement role management APIs
- [ ] Implement query execution engine
- [ ] Implement approval workflows
- [ ] Implement audit logging

### Phase 2: Advanced Features
- [ ] Multi-database support
- [ ] Schema discovery and caching
- [ ] Query result export (CSV, JSON)
- [ ] Saved query scripts
- [ ] Email/Slack notifications

### Phase 3: Enterprise Features
- [ ] SSO integration (SAML, OAuth)
- [ ] Multi-factor authentication
- [ ] Data masking for PII
- [ ] Query scheduling
- [ ] Collaborative queries
- [ ] AI-powered query suggestions

---

## 🤝 Contributing

### Documentation Updates
1. Update the relevant documentation file
2. Ensure examples are accurate and tested
3. Update the changelog
4. Submit a pull request

### Code Contributions
1. Review the [Backend Requirements](./backend-requirements.md)
2. Follow the API contracts in [API Documentation](./api-documentation.md)
3. Write tests for new features
4. Update documentation as needed

---

## 📞 Support

- **Documentation**: This directory
- **Issues**: GitHub Issues
- **Email**: support@sessiondb.com (if applicable)
- **Status**: https://status.sessiondb.com (if applicable)

---

## 📄 License

[Add your license information here]

---

## 📝 Changelog

### 2024-02-06
- ✅ Created comprehensive backend requirements document
- ✅ Created complete API documentation
- ✅ Documented all core features and workflows
- ✅ Added security and compliance specifications

---

**Last Updated**: February 6, 2024  
**Version**: 1.0.0  
**Maintained By**: SessionDB Team
