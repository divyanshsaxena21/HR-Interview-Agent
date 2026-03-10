# 🎯 VoxHire AI — Smart Voice Hiring Assistant

Welcome to **VoxHire AI** — your friendly, AI-powered recruiting assistant that runs natural voice interviews, transcribes responses, and provides structured evaluations for hiring teams. 🌟

Quick highlights:

- 🎤 Voice-first interviews (candidate speaks naturally)
- 🧠 AI-driven question generation & follow-ups
- 📊 Live evaluation: communication, technical, confidence, problem-solving
- ⚡️ Dev-friendly: runs locally without Redis (in-memory queue fallback)

---

## 🚀 Quick start (dev)

Prerequisites:

- Go 1.18+
- Node.js 18+
- MongoDB (cloud or local)

Start backend:

```bash
cd backend
# copy config: cp .env.example .env  (edit .env)
go run main.go
```

Start frontend:

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:3000 in your browser. ✨

---

## 🔐 Environment variables

Create a `backend/.env` with these (examples):

- `MONGO_URI` — MongoDB connection string
- `JWT_SECRET` — JWT signing secret
- `GROQ_API_KEY`, `GROQ_API_URL` — optional LLM provider
- `MURF_API_KEY` — optional Murf TTS
- `ASSEMBLYAI_API_KEY` — optional AssemblyAI STT
- `REDIS_URL` — optional; leave empty to use the in-memory evaluation queue

> Tip: For local development you can leave `REDIS_URL` empty — the app will automatically use an in-memory queue so you don't need Docker or an external Redis instance. 🧩

---

## 🧭 Project layout

- `backend/` — Go server, routes, controllers, services
- `frontend/` — Next.js app (App Router), components, pages
- `backend/services/eval_queue.go` — evaluation queue (Redis or in-memory)

---

## 🛠 Useful commands

- Build backend:

```bash
cd backend
go build ./...
```

- Run backend:

```bash
go run main.go
```

- Run frontend:

```bash
cd frontend
npm install
npm run dev
```

---

## 🧪 Features & flow (short)

1. Candidate enters session → AI asks first question
2. Candidate answers by voice → browser records audio
3. Audio → AssemblyAI (transcript) → backend
4. Backend uses LangChain/GROQ (or fallback templates) to generate follow-ups
5. AI question → Murf TTS → frontend plays audio
6. After each turn, evaluation job enqueues (in-memory or Redis) and analytics update live

---

## 📦 Evaluation scoring

Live metrics include:

- Communication: clarity & conciseness
- Technical knowledge: domain keywords & depth
- Confidence: hedging language vs assertiveness
- Problem-solving: structure and solution approach

All scores are stored in MongoDB and surfaced in the dashboard.

---

## ✨ Visuals & branding

You can style the frontend and README further to match your brand. Use emojis, badges, and screenshots to make it pop! If you'd like, I can add a simple SVG badge and a demo screenshot next.

---

## 🧾 Notes for contributors

- Keep secrets out of git — `.gitignore` already excludes `.env` and `node_modules`.
- For local testing, `REDIS_URL` may be empty (in-memory queue active).

---

If you want this README to be even more colorful (badges, GIF demo, or a marketing-style header), tell me your preferred tone (playful, corporate, modern) and I'll update it. 🎨


