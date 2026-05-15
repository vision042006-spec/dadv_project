# DADV – Metadata Analysis & Visualization Dashboard

A production-ready system for analyzing file metadata and generating insights through visualizations.

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new/template)

## 🚀 Live Demo

| Service  | URL |
|----------|-----|
| Frontend | Deployed on Netlify |
| API      | Deployed on Railway |

## ✨ Features

- **Drag-and-drop** CSV / JSON / Excel upload
- **Async processing** via Redis queue + Python worker
- **Smart Analysis**: file-type distribution, size buckets, temporal trends, ownership stats
- **Anomaly Detection**: PII scanning (emails, phones), size outliers
- **JWT Authentication** with user-scoped data isolation
- **Premium UI**: glassmorphism, smooth animations, responsive layout

---

## 🖥 Local Development

### Prerequisites
- Go 1.22+
- Python 3.11+
- Node.js 20+
- Docker (for Redis)

```bash
# Clone
git clone https://github.com/vision042006-spec/dadv_project.git
cd dadv_project

# Start Redis
docker run -d -p 6379:6379 redis:7-alpine

# Backend
go run ./cmd/api/main.go

# Worker (new terminal)
cd cmd/worker && pip install -r requirements.txt && python worker.py

# Frontend (new terminal)
cd frontend && npm install && npm run dev
```

Open http://localhost:5173

---

## 🌐 Production Deployment

### Backend (Railway)

1. Create a new Railway project and connect this repo
2. Add a **Redis** plugin in Railway
3. Set environment variables on the **API** service:
   ```
   REDIS_ADDR=<from railway redis plugin>
   REDIS_PASSWORD=<from railway redis plugin>
   DATABASE_DSN=/app/data/dadv.db
   JWT_SECRET=<generate a strong random string>
   CORS_ALLOWED_ORIGINS=https://your-app.netlify.app
   GIN_MODE=release
   ```
4. Add the same Redis env vars to the **Worker** service, plus:
   ```
   DATABASE_PATH=/app/data/dadv.db
   QUEUE_NAME=metadata_jobs
   ```
5. Mount a **shared volume** at `/app/data` for both the API and Worker services

### Frontend (Netlify)

1. Connect your GitHub repo to Netlify
2. Build settings (auto-detected from `netlify.toml`):
   - Base dir: `frontend`
   - Build command: `npm run build`
   - Publish dir: `frontend/dist`
3. Set environment variable:
   ```
   VITE_API_URL=https://your-api.railway.app
   ```

---

## 📁 Project Structure

```
dadv_project/
├── cmd/
│   ├── api/          # Go REST API (Gin)
│   └── worker/       # Python worker (Pandas + Redis)
├── frontend/         # React + Tailwind + Framer Motion
├── internal/         # Go packages (auth, db, handlers, middleware)
├── docker/           # Dockerfiles & docker-compose
├── railway.toml      # Railway deploy config
└── netlify.toml      # Netlify deploy config
```

## 🔐 Security

- JWT-based authentication on all data endpoints
- User-scoped data isolation
- File type allowlist + size limits
- Rate limiting (100 req/min per IP)
- CORS restricted to configured origins
- Input sanitization on all query params

## 📡 API Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | /api/auth/signup | — | Register |
| POST | /api/auth/login | — | Login |
| POST | /api/upload | ✅ | Upload dataset |
| GET | /api/job-status/:id | ✅ | Check job status |
| GET | /api/stats/aggregate/:id | ✅ | Aggregate stats |
| GET | /api/stats/file-types/:id | ✅ | File type breakdown |
| GET | /api/stats/size-distribution/:id | ✅ | Size buckets |
| GET | /api/anomalies/:id | ✅ | Detected anomalies |