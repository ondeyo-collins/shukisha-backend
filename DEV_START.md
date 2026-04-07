# Shukisha — Dev Start Guide

## Prerequisites

- Docker Desktop running
- Go 1.25+
- Node.js 20+

---

## 1. Start Infrastructure (PostgreSQL + Redis)

```powershell
# From the backend directory
docker compose up -d
```

Expected output:

```
✔ Container shukisha_db    Started
✔ Container shukisha_redis Started
```

---

## 2. Start the Go API Server

```powershell
# From the backend directory
go run ./cmd/api
```

The API will be available at **`http://localhost:8080`**

> ⚠️ Make sure `.env` exists (copy from `.env.example` if missing):
>
> ```powershell
> Copy-Item .env.example .env
> ```

---

## 3. Seed a SUPER_ADMIN (one-time)

The API does **AutoMigrate** on startup, but it does **not** automatically create an admin user.

Run this once to create a default SUPER_ADMIN:

```powershell
# From the backend directory
go run ./scripts/seed_superadmin.go
```

This seeds:
- Email: `superadmin@shukisha.com`
- Password: `admin123`

---

## 3. Start the Admin UI (SvelteKit)

```powershell
# From the admin-ui directory
cd admin-ui
npm run dev -- --port 5180
```

The admin dashboard will be available at **`http://localhost:5180`**

---

## 4. Full Stack — One Terminal Each

| Terminal                 | Command                                       | URL                |
| ------------------------ | --------------------------------------------- | ------------------ |
| **Infrastructure** | `docker compose up -d`                      | —                 |
| **API Server**     | `go run ./cmd/api`                          | `localhost:8080` |
| **Admin UI**       | `cd admin-ui && npm run dev -- --port 5180` | `localhost:5180` |

---

## 5. Stopping Everything

```powershell
# Stop Docker containers (from project root)
docker compose down

# Stop Go server → Ctrl+C in its terminal
# Stop Admin UI → Ctrl+C in its terminal
```

---

## Key Environment Variables (`.env`)

| Variable       | Description                             |
| -------------- | --------------------------------------- |
| `DB_HOST`    | PostgreSQL host (default:`localhost`) |
| `DB_PORT`    | PostgreSQL port (default:`5432`)      |
| `REDIS_HOST` | Redis host (default:`localhost`)      |
| `JWT_SECRET` | Secret for signing admin JWTs           |
| `TWILIO_*`   | SMS OTP credentials                     |
| `PORT`       | API server port (default:`8080`)      |

---

## Login Flow (Admin UI)

1. Navigate to `http://localhost:5180/login`
2. Sign in via Email + Password (recommended for admin dashboard)
3. You'll be redirected to the dashboard based on your role

> 💡 **SUPER_ADMIN** sees the full Dashboard pill with access to all departments.
> Department users (LOGISTICS, TRANSPORT, FINANCE, SUPPORT) only see their own pill.
