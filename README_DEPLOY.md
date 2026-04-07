# SHUKISHA Admin Dashboard - Deployment Guide

This document provides instructions for deploying the Shukisha Admin Dashboard (Backend API & Frontend UI) to a production environment.

## 1. System Requirements
- **Backend**: Go 1.21+
- **Frontend**: Node.js 20+ (SvelteKit)
- **Database**: PostgreSQL 15+ (with PostGIS extension for routing)
- **Cache**: Redis 7+ (for OTP and session management)

## 2. Environment Configuration
Copy the template and fill in your production credentials:
```bash
cp .env.example .env
```

### Essential Variables:
| Variable | Description | Example |
|----------|-------------|---------|
| `DB_HOST` | Postgres Host | `db.production.com` |
| `JWT_SECRET` | Secure Signing Key | `REPLACE_WITH_LONG_RANDOM_STRING` |
| `VITE_API_BASE_URL` | Frontend -> Backend URL | `https://api.shukisha.com/v1` |

## 3. Backend Deployment (Go)
1. **Build the binary**:
   ```bash
   go build -o shukisha-api ./cmd/api
   ```
2. **Run as a service** (e.g., using systemd or PM2):
   ```bash
   ./shukisha-api
   ```
   *Note: Ensure the `.env` file is in the same directory as the binary.*

## 3b. Backend Deployment (Docker Compose - Recommended)
This runs Postgres (with PostGIS + pgcrypto), Redis, the API, and the worker on one server.

```bash
cd backend
cp .env.example .env
# edit .env (JWT_SECRET, etc)
docker compose up -d --build
```

Notes:
- For Docker Compose, keep `DB_SSLMODE=disable` (the included Postgres container does not use SSL by default).

Health check:
```bash
curl http://localhost:8080/health
```

## 4. Frontend Deployment (SvelteKit)
1. **Install dependencies**:
   ```bash
   cd admin-ui && npm install
   ```
2. **Build for production**:
   ```bash
   npm run build
   ```
3. **Deploy the output**:
   - The build output is located in `admin-ui/build`.
   - You can host this on **Vercel**, **Netlify**, or use `adapter-node` to run it on your own server.

## 5. Production Checklist
- [ ] **SSL/TLS**: Ensure both API and UI are served over HTTPS.
- [ ] **CORS**: Update `internal/platform/middleware/cors.go` to restrict `Access-Control-Allow-Origin` to your production domain instead of `*`.
- [ ] **Secrets**: Change the default `JWT_SECRET` and database passwords.
- [ ] **Monitoring**: Set up logging (the application logs to stdout by default).

---
*Developed for Shukisha Administrative Infrastructure v1.0.0-mvp*
