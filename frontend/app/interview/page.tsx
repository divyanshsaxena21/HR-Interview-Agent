'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import VoiceRecorder from '@/components/VoiceRecorder'
import AudioPlayer from '@/components/AudioPlayer'
import TranscriptViewer from '@/components/TranscriptViewer'
import InterviewControls from '@/components/InterviewControls'

interface Message {
  role: string
  content: string
  timestamp: number
}

interface InterviewState {
  interviewId: string
  transcript: Message[]
  currentQuestion: string
  audioUrl: string
  isRecording: boolean
  questionCount: number
}

export default function InterviewPage() {
  const params = useParams()
  const interviewId = params.id as string
  const [state, setState] = useState<InterviewState>({
    interviewId,
    transcript: [],
    currentQuestion: 'Loading...',
    audioUrl: '',
    isRecording: false,
    questionCount: 0,
  })

  useEffect(() => {
    const loadInitialQuestion = async () => {
      try {
        const response = await fetch(
          `${process.env.NEXT_PUBLIC_API_URL}/interview/${interviewId}`
        )
        if (response.ok) {
          const data = await response.json()
          setState((prev) => ({
            ...prev,
            transcript: data.interview.transcript || [],
            currentQuestion: data.interview.current_question || prev.currentQuestion || 'Tell me about yourself.',
            audioUrl: data.interview.audio_url || '',
          }))
        }
      } catch (error) {
        console.error('Error loading interview:', error)
      }
    }

    loadInitialQuestion()
  }, [interviewId])

  // Fallback: if no audio URL is available, use browser SpeechSynthesis to speak the question
  useEffect(() => {
    if (!state.audioUrl && state.currentQuestion) {
      try {
        const utter = new SpeechSynthesisUtterance(state.currentQuestion)
        utter.rate = 1
        utter.pitch = 1
        window.speechSynthesis.cancel()
        window.speechSynthesis.speak(utter)
      } catch (e) {
        // ignore if SpeechSynthesis not available
      }
    }
  }, [state.currentQuestion, state.audioUrl])
  const handleTranscriptSubmit = async (transcript: string) => {
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/interview/chat`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            interview_id: interviewId,
            transcript,
          }),
        }
      )

      if (response.ok) {
        const data = await response.json()
        setState((prev) => ({
          ...prev,
          currentQuestion: data.question,
          audioUrl: data.audio_url,
          questionCount: prev.questionCount + 1,
          transcript: [
            ...prev.transcript,
            { role: 'candidate', content: transcript, timestamp: Date.now() },
            { role: 'ai', content: data.question, timestamp: Date.now() },
          ],
        }))
      }
    } catch (error) {
      console.error('Error sending transcript:', error)
    }
  }

  const handleFinishInterview = async () => {
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/interview/finish`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ interview_id: interviewId }),
        }
      )

      if (response.ok) {
        window.location.href = `/interview/${interviewId}`
      }
    } catch (error) {
      console.error('Error finishing interview:', error)
    }
  }

  return (
    <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
      <div className="max-w-4xl mx-auto">
        <div className="bg-white rounded-lg shadow-lg p-8 mb-6">
          <h1 className="text-3xl font-bold text-gray-800 mb-4">Interview</h1>

          <div className="mb-6 p-4 bg-gray-50 rounded-lg">
            <h2 className="text-lg font-semibold text-gray-700 mb-2">
              Current Question:
            </h2>
            <p className="text-gray-600 text-lg">{state.currentQuestion}</p>
          </div>

          {state.audioUrl && (
            <div className="mb-6">
              <AudioPlayer src={state.audioUrl} />
            </div>
          )}

          <div className="mb-6">
            <VoiceRecorder
              onTranscriptReady={handleTranscriptSubmit}
              disabled={state.questionCount >= 8}
            />
          </div>

          <TranscriptViewer messages={state.transcript} />

          <InterviewControls
            onFinish={handleFinishInterview}
            canFinish={state.questionCount > 2}
            questionsAsked={state.questionCount}
          />
        </div>
      </div>
    </main>
  )
}
