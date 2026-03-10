
# 🎤 VoxHire AI
### Smart Voice-Powered Hiring Assistant

**VoxHire AI** is an AI-driven recruiter that conducts **voice interviews directly in the browser**, analyzes candidate responses, and generates structured hiring insights for recruiters.

It simulates a **real screening interview** using AI — asking questions, listening to responses, and evaluating communication and technical ability.

---

## ✨ Features

🎤 **Voice-First Interviews**  
Candidates answer naturally using their microphone.

🧠 **AI Recruiter Agent**  
Dynamic interview questions with intelligent follow-ups.

📝 **Automatic Transcription**  
Speech-to-text powered by AssemblyAI.

🔊 **AI Voice Questions**  
Text-to-speech generated using Murf AI.

📊 **Candidate Evaluation Dashboard**

Live scoring across:

- Communication
- Technical Knowledge
- Confidence
- Problem Solving

⚡ **Developer Friendly**

- Runs locally
- No Redis required (in-memory queue fallback)

---

## 🧱 Architecture

```

Browser (Next.js)
│
▼
Go Backend (Gin)
│
▼
LangChain Agent
│
▼
Groq LLM
│
┌─────┴─────┐
▼           ▼
Murf TTS   AssemblyAI STT

````

---

## 🚀 Quick Start

### 1️⃣ Start Backend

```bash
cd backend
cp .env.example .env
go run main.go
````

---

### 2️⃣ Start Frontend

```bash
cd frontend
npm install
npm run dev
```

---

### 3️⃣ Open Application

```
http://localhost:3000
```

Start your AI-powered interview 🚀

---

## 🔐 Environment Variables

Create `backend/.env`

```env
MONGO_URI=

GROQ_API_KEY=
GROQ_API_URL=

MURF_API_KEY=
ASSEMBLYAI_API_KEY=

REDIS_URL=
```

Redis is **optional**.
If not provided, VoxHire automatically uses an **in-memory evaluation queue**.

---

## 📂 Project Structure

```
backend/
 ├─ controllers/
 ├─ routes/
 ├─ services/
 ├─ models/
 └─ main.go

frontend/
 ├─ app/
 ├─ components/
 ├─ lib/
 └─ next.config.js
```

---

## 🧪 Interview Flow

1. Candidate starts interview
2. AI asks question via voice
3. Candidate answers using microphone
4. Audio → transcript using AssemblyAI
5. LangChain + Groq generates follow-up question
6. Murf converts response to speech
7. Evaluation and analytics update live

---

## 📊 Candidate Evaluation

Each interview generates structured insights:

| Metric              | Description               |
| ------------------- | ------------------------- |
| Communication       | Clarity & conciseness     |
| Technical Knowledge | Domain understanding      |
| Confidence          | Assertiveness in answers  |
| Problem Solving     | Logical reasoning ability |

All interview data and analytics are stored in **MongoDB**.

---

## 🌟 Tech Stack

**Frontend**

* Next.js
* TypeScript
* TailwindCSS

**Backend**

* Go
* Gin
* MongoDB

**AI**

* LangChain
* Groq LLM
* Murf AI (TTS)
* AssemblyAI (STT)

---

## 💡 Why VoxHire?

VoxHire demonstrates how **AI agents can automate first-round hiring interviews**, enabling recruiters to evaluate candidates faster with structured insights and analytics.

---
