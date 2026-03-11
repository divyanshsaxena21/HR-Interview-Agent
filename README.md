# 💬 VoxHire AI
### Smart AI-Powered Hiring Assistant

**VoxHire AI** is an AI-driven recruiting platform that conducts **real-time text-based interviews**, evaluates candidates automatically, and provides structured insights for hiring teams.

Instead of manual screening calls, VoxHire uses an **AI recruiter agent** that chats with candidates, asks role-based questions, collects profile information, detects dealbreakers, and generates evaluation reports for HR teams.

---

## ✨ Features

💬 **Real-Time AI Interviews**  
Candidates interact with an AI recruiter through a live chat interface.

⚡ **Socket-Based Chat System**  
Real-time messaging powered by **Socket.IO**.

🧠 **AI Recruiter Agent**  
Dynamic questions and intelligent follow-ups generated using **LangChain + Groq**.

📚 **HR Memory Base**  
Recruiters can create and manage a question bank that the AI agent uses during interviews.

🚫 **Dealbreaker Detection**  
Automatic rejection for conditions like relocation refusal or other HR-defined dealbreakers.

📄 **Candidate Document Collection**

Candidates can upload:

- Resume
- Portfolio
- Supporting documents

All files are linked to the interview record.

🔗 **Candidate Profile Collection**

The AI agent collects candidate details such as:

- GitHub
- LinkedIn
- Portfolio

If not provided initially, the agent asks for them during the interview.

📊 **Recruiter Dashboard**

HR teams can view:

- Interview transcripts
- Candidate details
- Uploaded documents
- Evaluation scores
- Dealbreaker flags

🔐 **Admin Memory Editor**

HR administrators can:

- Add interview questions
- Edit existing questions
- Mark questions as dealbreakers
- Control which questions the AI asks

---

## 🧱 Architecture

```

Candidate Browser (Next.js)
│
▼
Socket.IO Chat System
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
▼
MongoDB

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

Start your AI-powered interview platform 🚀

---

## 🔐 Environment Variables

Create `backend/.env`

```env
MONGO_URI=

GROQ_API_KEY=
GROQ_API_URL=

JWT_SECRET=
PORT=8080
```

Frontend `.env.local`

```env
NEXT_PUBLIC_API_URL=
NEXT_PUBLIC_SOCKET_URL=
```

---

## 📂 Project Structure

```
backend/
 ├─ controllers/
 ├─ routes/
 ├─ services/
 ├─ models/
 ├─ sockets/
 └─ main.go

frontend/
 ├─ app/
 ├─ components/
 ├─ lib/
 └─ next.config.js
```

---

## 🧪 Interview Flow

1. Candidate starts interview session
2. AI recruiter sends first question via chat
3. Candidate responds through the chat interface
4. LangChain agent generates follow-up questions
5. AI collects missing details like GitHub or LinkedIn if not provided
6. Candidate can upload documents during the interview
7. System evaluates answers and detects dealbreaker conditions
8. Interview results are stored and displayed in the recruiter dashboard

---

## 📊 Candidate Evaluation

Each interview produces structured insights:

| Metric              | Description               |
| ------------------- | ------------------------- |
| Communication       | Clarity and conciseness   |
| Technical Knowledge | Domain understanding      |
| Confidence          | Assertiveness in answers  |
| Problem Solving     | Logical reasoning ability |

If a **dealbreaker condition** is triggered, the candidate is marked as **Rejected internally** without notifying the candidate.

---

## 🧠 HR Memory System

HR teams maintain a **question memory base** used by the AI recruiter.

Each question can include:

* Category
* Tags
* Active status
* Dealbreaker flag

This allows HR teams to dynamically control how interviews are conducted without changing code.

---

## 🌟 Tech Stack

**Frontend**

* Next.js
* TypeScript
* TailwindCSS
* Socket.IO Client

**Backend**

* Go
* Gin
* Socket.IO
* MongoDB

**AI**

* LangChain
* Groq LLM

---

## 💡 Why VoxHire?

VoxHire demonstrates how **AI agents can automate the initial hiring process**, allowing recruiters to screen candidates faster, collect structured insights, and focus only on the most promising applicants.

