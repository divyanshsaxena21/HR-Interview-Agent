# VoxHire AI Refactoring - Text-Based Chat Interview System

## Overview

VoxHire AI has been completely refactored from a **voice-based interview system** using Murf TTS and AssemblyAI STT to a **text-based real-time chat interview system** using Socket.IO, Groq LLM, and LangChain.

## Key Changes

### Backend Changes

#### 1. Removed Voice-Related Services
- Deleted Murf TTS API integration
- Deleted AssemblyAI STT API integration
- Removed audio recording and playback logic

#### 2. Updated Models (`backend/models/`)
- **interview.go**: Updated `Interview` model
  - Replaced `AudioURL` field with text-based fields
  - Added: `Email`, `GitHub`, `LinkedIn`, `Portfolio`, `Documents`
  - Renamed `Transcript` to `Messages` for clarity
  - Added: `Rejected`, `RejectionReason` for dealbreaker handling

- **admin.go**: Unchanged (existing JWT authentication)

- **hr_memory.go**: NEW
  - `HRMemory` model for question library
  - Schema: `category`, `question`, `tags`, `is_dealbreaker`, `active`
  - `HRMemoryRequest` for CRUD operations

#### 3. New Services (`backend/services/`)
- **hr_memory_service.go**: Manages question library
  - `CreateQuestion()`, `GetAllQuestions()`, `GetQuestionsByCategory()`
  - `UpdateQuestion()`, `DeleteQuestion()`
  
- **chat_service.go**: Handles real-time chat logic
  - `ProcessMessage()`: Sends candidate messages to Groq via LangChain
  - `SaveMessage()`: Persists messages to MongoDB
  - `CheckDealbreaker()`: Detects dealbreaker answers
  - `MarkAsRejected()`: Flags candidates with dealbreaker failures

- **websocket_service.go**: Socket.IO server
  - `WebSocketServer`: Manages client connections
  - Handles `candidate_message` and `ai_message` events
  - Broadcasts responses back to candidate

#### 4. Updated Services
- **langchain_agent.go**:
  - Added `GenerateResponse()`: Integrates with Groq API for text responses
  - Uses `mixtral-8x7b-32768` model via Groq
  - System prompt includes context about candidate info collection and dealbreakers

#### 5. New Controllers (`backend/controllers/`)
- **hr_memory_controller.go**: Manages interview question library
  - Admin endpoints for CRUD operations on questions
  - Protected by JWT middleware

#### 6. Updated Controllers
- **interview_controller.go**:
  - `StartInterview()`: Creates interview without TTS
  - `GetInterview()`: Returns interview data (candidate info + messages)
  - `UpdateCandidateInfo()`: Stores GitHub, LinkedIn, Portfolio
  - `EndInterview()`: Marks interview as completed
  - Removed: `UploadAudio()`, `Chat()`, `FinishInterview()`

- **admin_controller.go**:
  - `Login()`: JWT authentication (unchanged)
  - `GetAllInterviews()`: NEW - List all interviews for dashboard

#### 7. New Middleware (`backend/middleware/`)
- **auth.go**: JWT authentication middleware
  - Validates `Bearer {token}` header
  - Decodes claims for admin verification

#### 8. Updated Routes (`backend/routes/`)
- **interview_routes.go**:
  - `POST /interview/start`: Start new interview
  - `GET /interview/:id`: Get interview data
  - `PUT /interview/:id/info`: Update candidate info
  - `POST /interview/:id/end`: End interview

- **hr_memory_routes.go**: NEW
  - `GET /hr-memory/category/:category`: Get questions by category (public)
  - Protected routes:
    - `POST /admin/hr-memory`: Create question
    - `GET /admin/hr-memory`: Get all questions
    - `PUT /admin/hr-memory/:id`: Update question
    - `DELETE /admin/hr-memory/:id`: Delete question

- **admin_routes.go**:
  - `POST /admin/login`: Admin authentication
  - `GET /admin/interviews`: Get all interviews (protected)

#### 9. Environment Variables
- **Removed**:
  - `MURF_API_KEY`
  - `ASSEMBLYAI_API_KEY`

- **Added/Updated**:
  - `GROQ_API_KEY`: Groq API key for LLM
  - `JWT_SECRET`: Secret for JWT tokens
  - `PORT`: Server port (default: 8080)

### Frontend Changes

#### 1. Removed Components
- `VoiceRecorder.tsx`: Web Speech API integration removed
- `AudioPlayer.tsx`: Audio playback removed

#### 2. New Components (`frontend/components/`)
- **ChatInterface.tsx**: NEW
  - Real-time chat UI using Socket.IO
  - Message bubbles for candidate and AI
  - Auto-scrolling message list
  - Input field with Send button
  - Loading indicator for AI responses
  - Responsive design with Tailwind CSS

- **CandidateInfoForm.tsx**: NEW (replaces old form)
  - Collects: `name`, `email`, `role`
  - Simple, professional UI
  - Starts interview session

#### 3. Updated Pages (`frontend/app/`)
- **page.tsx** (Home):
  - Uses `CandidateInfoForm` component
  - Link to admin login in footer

- **interview/page.tsx**: NEW
  - Loads interview data
  - Displays `ChatInterface`

- **interview/[id]/page.tsx**: NEW
  - Same as interview/page.tsx
  - Alternative route structure

- **admin/login/page.tsx**: Updated
  - Uses axios for API calls
  - Stores token and admin name in localStorage

- **admin/dashboard/page.tsx**: Completely redesigned
  - Two tabs: "Interviews" and "Question Library"
  - Interview tab: Lists all interviews with details
  - Question Library tab: CRUD interface for questions
  - Add question form with: category, question text, tags, dealbreaker flag
  - Delete questions with confirmation
  - Logout functionality

#### 4. Dependencies Updates
- **package.json**:
  - Added: `socket.io-client` (^4.5.0)
  - Kept: `axios`, `react`, `next`

#### 5. Environment Variables
- **Frontend .env.local.example**:
  ```
  NEXT_PUBLIC_API_URL=http://localhost:8080
  NEXT_PUBLIC_SOCKET_URL=http://localhost:8080
  ```

## Interview Flow (Text-Based Chat)

### Candidate Flow
1. **Interview Start**: Candidate enters name, email, role
   - `POST /interview/start` creates new interview
   - Returns `interview_id` and Socket connection info
   
2. **Socket Connection**: Client connects via Socket.IO
   - Query params: `interviewId`, `candidateName`
   
3. **Chat Loop**:
   - Candidate types message → `candidate_message` event
   - Backend receives message → saves to MongoDB
   - LangChain agent generates response using Groq
   - Response saved → sent via `ai_message` event
   - UI updates with new message
   
4. **Dealbreaker Check**:
   - If answer indicates failure on dealbreaker question
   - Candidate marked as rejected internally
   - Interview continues naturally (no indication to candidate)
   
5. **Interview End**:
   - `POST /interview/:id/end` marks as completed

### Admin Flow
1. **Login**: `POST /admin/login` with admin_id + password
   - JWT token returned and stored
   
2. **Dashboard Access**: `/admin/dashboard`
   - View all interviews with status
   - See rejected candidates and reasons
   - View candidate links (GitHub, LinkedIn)
   
3. **Question Management**: `/admin/hr-memory`
   - Add new interview questions
   - Set category, tags, dealbreaker flag
   - Edit existing questions
   - Delete questions
   - Mark questions as active/inactive

## Data Models

### Interview Document (MongoDB)
```json
{
  "_id": "ObjectId",
  "candidate_name": "string",
  "email": "string",
  "role": "string",
  "github": "string (optional)",
  "linkedin": "string (optional)",
  "portfolio": "string (optional)",
  "messages": [
    {
      "role": "candidate | ai",
      "content": "string",
      "timestamp": "int64"
    }
  ],
  "documents": ["array of file paths (optional)"],
  "status": "in_progress | completed",
  "rejected": "boolean",
  "rejection_reason": "string (optional)",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "evaluation_id": "ObjectId (optional)"
}
```

### HR Memory Document (MongoDB)
```json
{
  "_id": "ObjectId",
  "category": "string (general|technical|behavioral|etc)",
  "question": "string",
  "tags": ["array of strings"],
  "is_dealbreaker": "boolean",
  "active": "boolean",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

## Socket.IO Events

### Client → Server
- **candidate_message**: `{ interviewId, message }`
  - Emitted when candidate sends message

### Server → Client
- **ai_message**: `{ type, content, role, sender, timestamp }`
  - Sent when AI generates response

- **error**: `string`
  - Sent on error conditions

## API Endpoints Summary

### Interview
- `POST /interview/start` - Create interview
- `GET /interview/:id` - Get interview details
- `PUT /interview/:id/info` - Update candidate info
- `POST /interview/:id/end` - End interview

### Admin
- `POST /admin/login` - Admin authentication
- `GET /admin/interviews` - List all interviews (protected)

### HR Memory
- `GET /hr-memory/category/:category` - Get questions by category
- `POST /admin/hr-memory` - Create question (protected)
- `GET /admin/hr-memory` - Get all questions (protected)
- `PUT /admin/hr-memory/:id` - Update question (protected)
- `DELETE /admin/hr-memory/:id` - Delete question (protected)

## Setup Instructions

### Backend
1. Update `.env` with:
   ```
   GROQ_API_KEY=your_groq_key
   JWT_SECRET=your_secret
   MONGO_URI=your_mongodb_uri
   PORT=8080
   ```

2. Install/update Go dependencies:
   ```bash
   go mod tidy
   ```

3. Run server:
   ```bash
   go run main.go
   ```

### Frontend
1. Update `.env.local` with:
   ```
   NEXT_PUBLIC_API_URL=http://localhost:8080
   NEXT_PUBLIC_SOCKET_URL=http://localhost:8080
   ```

2. Install dependencies:
   ```bash
   npm install
   # or
   yarn install
   ```

3. Run development server:
   ```bash
   npm run dev
   ```

## Key Features Implemented

✅ **Text-based chat interface** with real-time Socket.IO
✅ **Groq LLM integration** for intelligent responses
✅ **HR Memory/Question Library** - Admin can curate questions
✅ **Dealbreaker handling** - Auto-reject on specific answers
✅ **Candidate info collection** - GitHub, LinkedIn, Portfolio
✅ **Admin dashboard** - View interviews and manage questions
✅ **JWT authentication** - Secure admin access
✅ **Message persistence** - All chat stored in MongoDB
✅ **Responsive UI** - Mobile-friendly with Tailwind CSS
✅ **Production-ready code** - Modular and well-structured

## Migration from Voice System

If migrating from the old voice system:
1. Old `audio_url` field will be null for new interviews
2. Old `transcript` field renamed to `messages`
3. New interviews start with empty messages array
4. Socket.IO replaces HTTP polling for chat
5. HR Memory replaces hardcoded questions

## Notes

- Password hashing should be added to admin authentication for production
- Consider adding rate limiting for API endpoints
- Socket.IO connections should be properly closed on disconnect
- Implement proper error handling and logging
- Add input validation and sanitization
- Consider adding interview recording/transcription for records

---

**Refactoring Date**: March 10, 2026
**Status**: Complete
