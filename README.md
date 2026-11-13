# Snip - URL Shortener with Click Analytics

Sign in, paste a long URL, get a short link. Anyone who opens it is redirected, and the owner sees click counts (with a 7-day chart).

**Stack:** React (Vite) · Go (Gin) · MongoDB · Redis

## Run it locally

Requirements: Go 1.22+, Node 18+, Docker (for MongoDB and Redis).

```bash


# 1. Backend
cd backend
cp .env.example .env        # then set JWT_SECRET to a long random string
go mod tidy                 # downloads dependencies, creates go.sum
go run .                    # http://localhost:8080

# 2. Frontend 
cd frontend
npm install
npm run dev                 # http://localhost:5173
```

