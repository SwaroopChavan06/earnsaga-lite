# Project Brief: Go Backend Service

## Overview
This is a Go-based backend service designed with a clean, domain-driven architecture. It handles user authentication, wallet management, offer tracking, and analytics. It uses PostgreSQL for data persistence and integrates with external services like PubScale.

## Directory Structure
- `cmd/server/`: Contains the main entry point (`main.go`) for the application.
- `internal/`: Contains the core business logic, separated by domain.
    - `admin/`: Analytics and administrative reporting.
    - `auth/`: JWT handling, Google OAuth verification, and authentication middleware.
    - `common/`: Shared utilities, specifically for HTTP response handling.
    - `config/`: Environment variable loading and configuration management.
    - `db/`: Database connection pooling using `pgx`.
    - `event/`: Event tracking logic (impressions, clicks).
    - `leaderboard/`: User ranking and earnings logic.
    - `models/`: Shared data structures (User, Offer, WalletTransaction, etc.).
    - `pubscale/`: Client for interacting with the external PubScale API.
    - `user/`: User profile management.
    - `wallet/`: Wallet balance and transaction history.
- `migrations/`: SQL migration files for database schema management.

## Key Technologies
- **Language:** Go (Golang)
- **Database:** PostgreSQL (via `pgx/v5` and `pgxpool`)
- **Authentication:** JWT (JSON Web Tokens), Google ID Token validation
- **Configuration:** `godotenv` for environment variables
- **Architecture:** Domain-driven design with a clear separation of Handlers (HTTP), Services (Business Logic), and Repositories (Data Access).

## Development Guidelines for AI
1. **Clean Architecture:** Always maintain the separation between layers. Handlers should only parse requests and call services. Services should contain business logic. Repositories should only handle SQL queries.
2. **Error Handling:** Use `common.WriteError` for consistent API error responses.
3. **Context:** Always pass `context.Context` through the layers to handle timeouts and cancellations.
4. **Database:** Use `pgxpool` for database operations. Ensure all queries are parameterized to prevent SQL injection.
5. **Models:** Shared structs are located in `internal/models`.
