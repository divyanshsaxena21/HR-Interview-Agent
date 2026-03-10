import axios from 'axios'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

export const startInterview = (data: {
  name: string
  email: string
  role: string
}) => api.post('/interview/start', data)

export const sendTranscript = (
  interviewId: string,
  transcript: string
) =>
  api.post('/interview/chat', {
    interview_id: interviewId,
    transcript,
  })

export const finishInterview = (interviewId: string) =>
  api.post('/interview/finish', {
    interview_id: interviewId,
  })

export const getInterview = (id: string) =>
  api.get(`/interview/${id}`)

export const getAllInterviews = () =>
  api.get('/interviews')

export default api
