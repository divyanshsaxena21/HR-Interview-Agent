"use client"

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import VoiceRecorder from '@/components/VoiceRecorder'
import TranscriptViewer from '@/components/TranscriptViewer'
import InterviewControls from '@/components/InterviewControls'
import AudioPlayer from '@/components/AudioPlayer'
import FitBadge from '@/components/FitBadge'

interface Evaluation {
  candidate_name: string
  role: string
  communication_score: number
  technical_score: number
  confidence_score: number
  problem_solving_score: number
  strengths: string[]
  weaknesses: string[]
  summary: string
  fit: string
}

interface Analytics {
  avg_answer_length: number
  followups_needed: number
  clarity_rating: number
  candidate_talk_ratio: number
}

interface Message {
  role: string
  content: string
  timestamp: number
}

interface InterviewData {
  _id: string
  candidate_name: string
  role: string
  status: string
  transcript: Message[]
  evaluation?: Evaluation
  analytics?: Analytics
}

export default function InterviewPage() {
  const params = useParams()
  const id = params.id as string

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

  const [interviewData, setInterviewData] = useState<InterviewData | null>(null)
  const [currentQuestion, setCurrentQuestion] = useState('')
  const [currentAudioUrl, setCurrentAudioUrl] = useState('')
  const [loading, setLoading] = useState(true)
  const [isSubmittingAnswer, setIsSubmittingAnswer] = useState(false)
  const [autoStartRecorder, setAutoStartRecorder] = useState(false)
  const [audioAutoPlay, setAudioAutoPlay] = useState(false)
  const [isAdmin, setIsAdmin] = useState(false)

  useEffect(() => {
    // detect admin token to allow admins to see evaluation and include it in requests
    let token: string | null = null
    try {
      token = typeof window !== 'undefined' ? localStorage.getItem('admin_token') : null
      setIsAdmin(!!token)
    } catch (e) {
      setIsAdmin(false)
    }

    const headers: Record<string, string> = {}
    if (token) headers['Authorization'] = `Bearer ${token}`

    const loadInterview = async () => {
      try {
        const response = await fetch(`${API_BASE}/interview/${id}`, { headers })
        if (response.ok) {
          const data = await response.json()
          console.log('Initial interview load:', data)
            setInterviewData({ ...(data.interview || {}), evaluation: data.evaluation, analytics: data.analytics })
            const q = data.currentQuestion || (data.interview && data.interview.current_question) || 'Could you start by introducing yourself and briefly summarizing your professional background relevant to this role?'
            setCurrentQuestion(q)
            const audio = (data.interview && (data.interview.audio_url || data.interview.audioUrl)) || ''
            setCurrentAudioUrl(audio)

            if (audio) {
              // autoplay server TTS for the initial question
              setAudioAutoPlay(true)
            } else {
              // If there's no server TTS audio, speak the question via SpeechSynthesis
              try {
                const utter = new SpeechSynthesisUtterance(q)
                window.speechSynthesis.cancel()
                window.speechSynthesis.speak(utter)
              } catch (e) {}
            }
        }
      } catch (error) {
        console.error('Error loading interview:', error)
      } finally {
        setLoading(false)
      }
    }

    loadInterview()
  }, [id])

  // Cooldown (ms) between agent audio end and starting the recorder
  const RECORD_COOLDOWN_MS = 900

  const speakText = (text: string) => {
    try {
      if (typeof window !== 'undefined' && 'speechSynthesis' in window) {
        const utterance = new SpeechSynthesisUtterance(text)
        window.speechSynthesis.cancel()
        window.speechSynthesis.speak(utterance)
      } else {
        console.log('SpeechSynthesis not supported in this browser.')
      }
    } catch (e) {
      console.error('SpeechSynthesis error:', e)
    }
  }

  const handleTranscriptReady = async (transcript: string) => {
    setIsSubmittingAnswer(true)
    setAutoStartRecorder(false)
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('admin_token') : null
      const postHeaders: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) postHeaders['Authorization'] = `Bearer ${token}`

      const response = await fetch(`${API_BASE}/interview/chat`, {
        method: 'POST',
        headers: postHeaders,
        body: JSON.stringify({ interview_id: id, transcript }),
      })

      if (!response.ok) {
        console.error('Chat response error:', response.status, response.statusText)
        return
      }

      const result = await response.json()
      console.log('Chat response:', result)

      if (result.finished) {
        // Merge evaluation/analytics into UI state
        setInterviewData((prev) => ({ ...(prev as InterviewData), status: 'completed', evaluation: result.evaluation || (prev as any).evaluation, analytics: result.analytics || (prev as any).analytics }))
        return
      }

      const nextQuestion = result.question || result.question_text || 'Thank you for your answer.'
      const audioFromServer = result.audio_url || result.audioURL || result.audio || result.audioUrl || ''

      setCurrentQuestion(nextQuestion)
      setCurrentAudioUrl(audioFromServer)

      // Reload interview to get updated transcript
      const token2 = typeof window !== 'undefined' ? localStorage.getItem('admin_token') : null
      const getHeaders: Record<string, string> = {}
      if (token2) getHeaders['Authorization'] = `Bearer ${token2}`

      const updatedResponse = await fetch(`${API_BASE}/interview/${id}`, { cache: 'no-store', headers: getHeaders })
      if (updatedResponse.ok) {
        const updatedData = await updatedResponse.json()
        console.log('Updated interview data:', updatedData)
        setInterviewData({ ...(updatedData.interview || {}), evaluation: updatedData.evaluation, analytics: updatedData.analytics })
      }

      if (audioFromServer) {
        setAudioAutoPlay(true)
      } else {
        speakText(nextQuestion)
        // ensure recorder has a matching grace period before starting silence timer
        if (typeof window !== 'undefined') {
          ;(window as any).VOICE_SILENCE_GRACE_MS = RECORD_COOLDOWN_MS
        }
        setTimeout(() => setAutoStartRecorder(true), RECORD_COOLDOWN_MS)
      }
    } catch (error) {
      console.error('Error submitting answer:', error)
    } finally {
      setIsSubmittingAnswer(false)
    }
  }

  const handleFinishInterview = async () => {
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('admin_token') : null
      const postHeaders: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) postHeaders['Authorization'] = `Bearer ${token}`

      const response = await fetch(`${API_BASE}/interview/finish`, {
        method: 'POST',
        headers: postHeaders,
        body: JSON.stringify({ interview_id: id }),
      })

      if (response.ok) {
        const result = await response.json()
        const updatedResponse = await fetch(`${API_BASE}/interview/${id}`)
        if (updatedResponse.ok) {
          const updatedData = await updatedResponse.json()
          setInterviewData({ ...(updatedData.interview || {}), evaluation: updatedData.evaluation, analytics: updatedData.analytics })
        } else {
          setInterviewData((prev) => ({ ...(prev as InterviewData), status: 'completed', evaluation: result.evaluation, analytics: result.analytics }))
        }
      }
    } catch (error) {
      console.error('Error finishing interview:', error)
    }
  }

  if (loading) {
    return (
      <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center">
        <div className="text-xl text-gray-700">Loading interview...</div>
      </main>
    )
  }

  if (!interviewData) {
    return (
      <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center">
        <div className="text-xl text-gray-700">Interview not found</div>
      </main>
    )
  }

  if (interviewData.status === 'completed') {
    // Only show evaluation/analytics to admins
    if (isAdmin && interviewData.evaluation) {
      const evaluation = interviewData.evaluation
      return (
        <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
          <div className="max-w-6xl mx-auto">
            <div className="flex items-center gap-4 mb-6">
              <a href="/" className="text-blue-600 hover:text-blue-800 inline-block">← Back to Home</a>
              {isAdmin && (
                <a href="/admin/dashboard" className="text-blue-600 hover:text-blue-800 inline-block">← Back to Dashboard</a>
              )}
            </div>

            <div className="bg-white rounded-lg shadow-lg p-8 mb-6">
              <div className="mb-6">
                <h1 className="text-4xl font-bold text-gray-800 mb-2">{evaluation.candidate_name}</h1>
                <p className="text-xl text-gray-600">Role: {evaluation.role}</p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
                <div className="p-4 bg-blue-50 rounded-lg">
                  <h3 className="font-semibold text-blue-900 mb-2">Communication Score</h3>
                  <p className="text-3xl font-bold text-blue-600">{evaluation.communication_score}/10</p>
                </div>
                <div className="p-4 bg-green-50 rounded-lg">
                  <h3 className="font-semibold text-green-900 mb-2">Technical Score</h3>
                  <p className="text-3xl font-bold text-green-600">{evaluation.technical_score}/10</p>
                </div>
                <div className="p-4 bg-purple-50 rounded-lg">
                  <h3 className="font-semibold text-purple-900 mb-2">Confidence Score</h3>
                  <p className="text-3xl font-bold text-purple-600">{evaluation.confidence_score}/10</p>
                </div>
                <div className="p-4 bg-orange-50 rounded-lg">
                  <h3 className="font-semibold text-orange-900 mb-2">Problem Solving Score</h3>
                  <p className="text-3xl font-bold text-orange-600">{evaluation.problem_solving_score}/10</p>
                </div>
              </div>

              <div className="mb-6 p-4 bg-yellow-50 rounded-lg border-2 border-yellow-200">
                <h3 className="font-semibold text-yellow-900 mb-2">Fit Status</h3>
                <p className="text-2xl font-bold"><FitBadge fit={evaluation.fit} /></p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
                <div>
                  <h3 className="font-semibold text-gray-800 mb-3">Strengths</h3>
                  <ul className="space-y-2">
                    {evaluation.strengths && evaluation.strengths.map((strength, i) => (
                      <li key={i} className="text-gray-700 flex items-start gap-2"><span className="text-green-600 font-bold">+</span>{strength}</li>
                    ))}
                  </ul>
                </div>
                <div>
                  <h3 className="font-semibold text-gray-800 mb-3">Weaknesses</h3>
                  <ul className="space-y-2">
                    {evaluation.weaknesses && evaluation.weaknesses.map((weakness, i) => (
                      <li key={i} className="text-gray-700 flex items-start gap-2"><span className="text-red-600 font-bold">-</span>{weakness}</li>
                    ))}
                  </ul>
                </div>
              </div>

              <div className="p-4 bg-gray-50 rounded-lg mb-6">
                <h3 className="font-semibold text-gray-800 mb-2">Summary</h3>
                <p className="text-gray-700">{evaluation.summary}</p>
              </div>
            </div>

            {interviewData.analytics && (
              <div className="bg-white rounded-lg shadow-lg p-8">
                <h2 className="text-2xl font-bold text-gray-800 mb-4">Analytics</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="p-4 bg-gray-50 rounded-lg">
                    <h3 className="font-semibold text-gray-800 mb-2">Average Answer Length</h3>
                    <p className="text-2xl font-bold text-gray-700">{interviewData.analytics.avg_answer_length} seconds</p>
                  </div>
                  <div className="p-4 bg-gray-50 rounded-lg">
                    <h3 className="font-semibold text-gray-800 mb-2">Clarity Rating</h3>
                    <p className="text-2xl font-bold text-gray-700">{interviewData.analytics.clarity_rating}/10</p>
                  </div>
                  <div className="p-4 bg-gray-50 rounded-lg">
                    <h3 className="font-semibold text-gray-800 mb-2">Followups Needed</h3>
                    <p className="text-2xl font-bold text-gray-700">{interviewData.analytics.followups_needed}</p>
                  </div>
                  <div className="p-4 bg-gray-50 rounded-lg">
                    <h3 className="font-semibold text-gray-800 mb-2">Candidate Talk Ratio</h3>
                    <p className="text-2xl font-bold text-gray-700">{(interviewData.analytics.candidate_talk_ratio * 100).toFixed(1)}%</p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </main>
      )
    }

    // not admin: show generic completion message only
    return (
      <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
        <div className="max-w-4xl mx-auto">
          <div className="bg-white rounded-lg shadow-lg p-8">
            <h2 className="text-2xl font-semibold text-gray-800 mb-4">Interview Completed</h2>
            <p className="text-gray-700">Thank you — this interview is complete. Results are available to administrators only.</p>
            <div className="mt-6">
              <a href="/" className="text-blue-600 hover:text-blue-800">← Back to Home</a>
            </div>
          </div>
        </div>
      </main>
    )
  }

  return (
    <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
      <div className="max-w-4xl mx-auto">
        <div className="flex items-center gap-4 mb-6">
          <a href="/" className="text-blue-600 hover:text-blue-800 inline-block">← Back to Home</a>
          {isAdmin && (
            <a href="/admin/dashboard" className="text-blue-600 hover:text-blue-800 inline-block">← Back to Dashboard</a>
          )}
        </div>

        <div className="bg-white rounded-lg shadow-lg p-8">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-gray-800 mb-2">{interviewData.candidate_name}</h1>
            <p className="text-lg text-gray-600 mb-4">Position: {interviewData.role}</p>
            <div className="h-1 bg-gradient-to-r from-blue-400 to-indigo-600 rounded"></div>
          </div>

          {/* Debug Panel */}
          <div className="mb-8 p-4 bg-gray-100 rounded-lg border border-gray-300 text-sm">
            <p className="font-semibold text-gray-800">Debug Info:</p>
            <p className="text-gray-700">Transcript Length: {interviewData.transcript?.length || 0}</p>
            <p className="text-gray-700">Questions Asked: {(interviewData.transcript?.filter((m) => m.role === 'ai').length) || 0}</p>
            <p className="text-gray-700">Interview Status: {interviewData.status}</p>
          </div>

          <div className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-800 mb-4">Question</h2>
            <div className="p-6 bg-blue-50 rounded-lg border-l-4 border-blue-600">
              <p className="text-lg text-gray-700">{currentQuestion}</p>
            </div>
            {currentAudioUrl && (
              <div className="mt-4">
                <AudioPlayer
                  src={currentAudioUrl}
                  autoPlay={audioAutoPlay}
                  onEnded={() => {
                    // Wait a short cooldown before starting the recorder so the browser
                    // can capture the candidate's first syllables and avoid immediate timeout
                    if (typeof window !== 'undefined') {
                      ;(window as any).VOICE_SILENCE_GRACE_MS = RECORD_COOLDOWN_MS
                    }
                    setTimeout(() => setAutoStartRecorder(true), RECORD_COOLDOWN_MS)
                    setAudioAutoPlay(false)
                  }}
                />
              </div>
            )}
          </div>

          <div className="mb-8">
            <h2 className="text-2xl font-semibold text-gray-800 mb-4">Your Answer</h2>
            <VoiceRecorder onTranscriptReady={handleTranscriptReady} disabled={isSubmittingAnswer} autoStart={autoStartRecorder} autoSubmit={true} />
          </div>

          {interviewData.transcript && interviewData.transcript.length > 0 && (
            <div className="mb-8">
              <h2 className="text-2xl font-semibold text-gray-800 mb-4">Interview Transcript</h2>
              <TranscriptViewer messages={interviewData.transcript} />
            </div>
          )}

          <InterviewControls onFinish={handleFinishInterview} canFinish={(interviewData.transcript?.length || 0) >= 3} questionsAsked={(interviewData.transcript?.filter((m) => m.role === 'ai' || m.role === 'assistant').length) || 0} />
        </div>
      </div>
    </main>
  )
}
