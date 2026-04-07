# Shukisha — Dev Start Guide

## Prerequisites

- Docker Desktop running
- Go 1.22+
- Node.js 20+

---

## 1. Start Infrastructure (PostgreSQL + Redis)

```powershell
# From the project root
docker-compose up -d
```

Expected output:

```
✔ Container shukisha_db    Started
✔ Container shukisha_redis Started
```

---

## 2. Start the Go API Server

```powershell
# From the project root
go run ./cmd/api
```

The API will be available at **`http://localhost:8080`**

> ⚠️ Make sure `.env` exists (copy from `.env.example` if missing):
>
> ```powershell
> Copy-Item .env.example .env
> ```

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
| **Infrastructure** | `docker-compose up -d`                      | —                 |
| **API Server**     | `go run ./cmd/api`                          | `localhost:8080` |
| **Admin UI**       | `cd admin-ui && npm run dev -- --port 5180` | `localhost:5180` |

---

## 5. Stopping Everything

```powershell
# Stop Docker containers (from project root)
docker-compose down

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
2. Enter your registered admin phone number
3. Enter the 6-digit OTP sent via SMS
4. You'll be redirected to the dashboard based on your role

> 💡 **SUPER_ADMIN** sees the full Dashboard pill with access to all departments.
> Department users (LOGISTICS, TRANSPORT, FINANCE, SUPPORT) only see their own pill.
>
> Successfully created SUPER_ADMIN:
> Email: admin@shukisha.co.ke
> Password: CHANGE_ME
