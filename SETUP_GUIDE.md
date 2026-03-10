# VoxHire AI - Text-Based Chat Interview System

## Quick Start Guide

### Prerequisites
- Go 1.21+
- Node.js 18+
- MongoDB Atlas or local MongoDB
- Groq API key (https://console.groq.com)

### Backend Setup

1. **Install dependencies**:
   ```bash
   cd backend
   go mod tidy
   ```

2. **Create `.env` file** from `.env.example`:
   ```bash
   cp .env.example .env
   ```

3. **Update `.env` with your credentials**:
   ```
   GROQ_API_KEY=your_groq_api_key_here
   MONGO_URI=mongodb+srv://user:password@cluster.mongodb.net/ai_recruiter
   PORT=8080
   JWT_SECRET=your_secure_secret_key_here
   ```

4. **(Optional) Create dev admin** - Add to `.env`:
   ```
   DEV_CREATE_ADMIN=true
   DEV_ADMIN_ID=admin
   DEV_ADMIN_PASS=admin123
   DEV_ADMIN_NAME=Admin User
   DEV_ADMIN_EMAIL=admin@voxhire.com
   ```

5. **Run backend**:
   ```bash
   go run main.go
   ```

   Server will start on `http://localhost:8080`

### Frontend Setup

1. **Install dependencies**:
   ```bash
   cd frontend
   npm install
   ```

2. **Create `.env.local` file**:
   ```
   NEXT_PUBLIC_API_URL=http://localhost:8080
   NEXT_PUBLIC_SOCKET_URL=http://localhost:8080
   ```

3. **Run development server**:
   ```bash
   npm run dev
   ```

   Frontend will start on `http://localhost:3000`

### Testing the Application

#### Candidate Interview Flow
1. Navigate to `http://localhost:3000`
2. Enter candidate info (name, email, role)
3. Click "Start Interview"
4. Type responses in the chat interface
5. AI will respond with interview questions
6. Chat continues in real-time via Socket.IO

#### Admin Dashboard
1. Navigate to `http://localhost:3000/admin/login`
2. Login with credentials (default: admin/admin123)
3. View interviews and interview transcripts
4. Manage interview question library
5. Add/edit/delete questions
6. Mark questions as dealbreakers

### Key Endpoints

**Candidate Interview**:
- `POST /interview/start` - Start new interview
- `GET /interview/:id` - Get interview details

**Admin**:
- `POST /admin/login` - Admin login
- `GET /admin/interviews` - List all interviews

**HR Memory (Questions)**:
- `GET /hr-memory/category/:category` - Get questions by category
- `POST /admin/hr-memory` - Add new question (admin only)
- `GET /admin/hr-memory` - Get all questions (admin only)
- `PUT /admin/hr-memory/:id` - Edit question (admin only)
- `DELETE /admin/hr-memory/:id` - Delete question (admin only)

### Architecture Overview

```
┌─────────────────────────────────────────┐
│         Frontend (Next.js)               │
│  - Candidate Interview Page              │
│  - Admin Dashboard                       │
│  - Real-time Chat (Socket.IO)            │
└────────────────┬────────────────────────┘
                 │
                 │ HTTP/WebSocket
                 │
┌────────────────▼────────────────────────┐
│         Backend (Go/Gin)                 │
│  - REST API Endpoints                    │
│  - Socket.IO Server                      │
│  - JWT Authentication                    │
│  - LangChain Agent Integration           │
└────────────────┬────────────────────────┘
                 │
                 │ mongoDB Queries
                 │
┌────────────────▼────────────────────────┐
│         MongoDB                          │
│  - interviews collection                 │
│  - hr_memory collection                  │
│  - admins collection                     │
│  - evaluations collection                │
└─────────────────────────────────────────┘
```

### System Workflow

**Interview Interview**:
1. Candidate fills in basic info
2. Interview record created in MongoDB
3. Socket connection established
4. Candidate sends message
5. Backend receives via Socket.IO
6. Message saved to MongoDB
7. Groq LLM generates response (via LangChain)
8. Response saved and sent back via Socket
9. Process repeats until interview ends

**Dealbreaker Detection**:
1. Admin marks question as dealbreaker
2. During interview, system checks answers
3. If dealbreaker answer detected
4. Candidate marked as rejected internally
5. Interview continues normally
6. Admin sees rejection status in dashboard

### Production Checklist

- [ ] Replace development admin credentials
- [ ] Use strong JWT_SECRET (generate with `openssl rand -base64 32`)
- [ ] Enable HTTPS/TLS
- [ ] Add password hashing (bcrypt) for admin credentials
- [ ] Implement rate limiting
- [ ] Add request validation/sanitization
- [ ] Set up proper logging/monitoring
- [ ] Configure CORS properly
- [ ] Use environment-specific configurations
- [ ] Set up database backups
- [ ] Add input validation on all endpoints
- [ ] Implement proper error handling
- [ ] Add analytics tracking (optional)

### Troubleshooting

**Socket.IO Connection Issues**:
- Ensure backend is running and accessible
- Check `NEXT_PUBLIC_SOCKET_URL` env variable
- Verify CORS settings in backend
- Check browser console for errors

**Groq API Errors**:
- Verify `GROQ_API_KEY` is correct
- Check Groq account quota
- Verify API key has appropriate permissions

**MongoDB Connection Issues**:
- Verify `MONGO_URI` is correct
- Check MongoDB Atlas firewall settings
- Ensure IP whitelist includes your IP
- Verify database credentials

**Admin Login Issues**:
- Verify `JWT_SECRET` is set
- Check admin credentials in database
- Clear browser localStorage and retry

### Development Tips

- Use `DEV_CREATE_ADMIN=true` for automatic admin creation
- Enable MongoDB logging for debugging
- Check Groq API response logs
- Monitor Socket.IO connections with `io.engine.clientsCount`
- Use Postman/Thunder Client for API testing

---

For more details, see `REFACTORING_NOTES.md`
