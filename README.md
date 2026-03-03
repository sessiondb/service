# SessionDB
**The Open Core Gateway for Granular Database Access**

SessionDB is a modern Database Management & Access Control Gateway inspired by tools like Redash, but built for dynamic multi-tenancy, granular Role-Based Access Control (RBAC), and high observability. 

It provides teams with a unified interface to query databases safely while strictly governing *who* can access *what data* and *for how long*.

---

## ⚡ Core Features vs. Pro Features

SessionDB follows an **Open Core** model. This repository contains the **Community Edition**, which provides all the foundational logic and security layers necessary to run a robust internal data gateway.

| Feature Category | Community Edition (This Repo) | Pro / Managed Service |
| :--- | :--- | :--- |
| **Data Access** | Interactive SQL Editor, Schema Viewer | **Query Insights**, Execution Analytics |
| **Access Control** | Basic RBAC (`users:read`, `users:write`) | **TTL-based Table Access**, Expiring Grants |
| **Security** | Static Database Credentials | **Auto Credentials Expiry** & Rotation |
| **Tenancy** | Multi-Tenancy Engine, Bill Splitting Core | **DB Metrics** & Performance Tracking |
| **Workflows** | Basic Approval Flow | **Advanced Approval Workflows**, Schema Alters |

> **Note:** Pro features are only available in the commercial or managed versions of SessionDB.

---

## 🚀 Quick Start

You can run the full Community Edition stack quickly using Docker Compose.

```bash
# 1. Clone the repository
git clone https://github.com/yourusername/sessiondb.git
cd sessiondb

# 2. Start the services (Go backend, React frontend, and PostgreSQL metadata DB)
docker compose up -d

# 3. Access the application
# The UI will be available at http://localhost:3000
# The API will be available at http://localhost:8080
```

---

## 🏗️ Architecture & Security Model

SessionDB is designed to be mathematically secure. Authentication states, RBAC permissions, and Feature Flags are all baked directly into secure JWTs. 

If you are contributing to the UI or API, please be aware of our strict access gates:
- **Backend (`CheckPermission`)**: API middleware that rejects HTTP requests if the user lacks the explicit permission string (e.g., `users:write`).
- **Backend (`FeatureGate`)**: API middleware that automatically 403s requests targeting premium commercial features, returning a `plan_upgrade_required` error.
- **Frontend (`<PermissionGate>`)**: React wrapper that hides UI components if the user lacks the correct RBAC capability.
- **Frontend (`<FeatureGate>`)**: React wrapper that displays "Upgrade to Pro" lock screens for advanced features.

---

## ⚖️ Licensing

SessionDB Community Edition is licensed under the **Business Source License (BSL) 1.1**.

**What you CAN do:**
- Use this software for private, personal projects.
- Use this software for internal business operations and testing.
- Fork, modify, and contribute to the source code.

**What you CANNOT do:**
- You are explicitly forbidden from using this source code to provide a competing commercial service (such as a Database-as-a-Service, Managed Query Environment, or commercial Data Observability Platform).

For full details, please read the [LICENSE](LICENSE) and [NOTICE](NOTICE) files. If you require a commercial license or wish to upgrade to the Pro tier, please contact us.
