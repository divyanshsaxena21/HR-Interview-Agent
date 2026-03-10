'use client'

import { useState, useRef, useEffect } from 'react'

interface VoiceRecorderProps {
  onTranscriptReady: (transcript: string) => void
  disabled?: boolean
  autoStart?: boolean
  autoSubmit?: boolean
  silenceTimeoutMs?: number
}

declare global {
  interface Window {
    SpeechRecognition: any
    webkitSpeechRecognition: any
  }
}

export default function VoiceRecorder({
  onTranscriptReady,
  disabled = false,
  autoStart = false,
  autoSubmit = true,
  silenceTimeoutMs = 5000,
}: VoiceRecorderProps) {
  const [isRecording, setIsRecording] = useState(false)
  const [isProcessing, setIsProcessing] = useState(false)
  const [transcript, setTranscript] = useState('')
  const transcriptRef = useRef('')
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const audioChunksRef = useRef<Blob[]>([])
  const recognitionRef = useRef<any>(null)
  const silenceTimerRef = useRef<number | null>(null)
  const silenceIntervalRef = useRef<number | null>(null)
  const graceTimeoutRef = useRef<number | null>(null)
  const [silenceRemainingSec, setSilenceRemainingSec] = useState<number | null>(null)

  useEffect(() => {
    // Initialize Web Speech API
    const SpeechRecognition = window.SpeechRecognition || (window as any).webkitSpeechRecognition
    if (SpeechRecognition) {
      recognitionRef.current = new SpeechRecognition()
      recognitionRef.current.continuous = true
      recognitionRef.current.interimResults = true
      recognitionRef.current.language = 'en-US'

      // Add a small grace period before starting the silence timeout so the
      // browser and microphone can stabilize after audio playback finishes.
      recognitionRef.current.onstart = () => {
        setTranscript('')
        const graceMs = (window as any).VOICE_SILENCE_GRACE_MS || 700
        // Clear any previous grace timers
        if (graceTimeoutRef.current) {
          window.clearTimeout(graceTimeoutRef.current)
          graceTimeoutRef.current = null
        }

        graceTimeoutRef.current = window.setTimeout(() => {
          // start silence timer if configured
          if (silenceTimeoutMs && silenceTimeoutMs > 0) {
            if (silenceTimerRef.current) {
              window.clearTimeout(silenceTimerRef.current)
            }
            if (silenceIntervalRef.current) {
              window.clearInterval(silenceIntervalRef.current)
            }
            // set remaining seconds for UI
            const totalSec = Math.ceil(silenceTimeoutMs / 1000)
            setSilenceRemainingSec(totalSec)
            // interval to update remaining seconds
            silenceIntervalRef.current = window.setInterval(() => {
              setSilenceRemainingSec((prev) => {
                if (prev === null) return null
                if (prev <= 1) {
                  // time up, clear interval
                  if (silenceIntervalRef.current) {
                    window.clearInterval(silenceIntervalRef.current)
                    silenceIntervalRef.current = null
                  }
                  return 0
                }
                return prev - 1
              })
            }, 1000)

            silenceTimerRef.current = window.setTimeout(() => {
              // no speech detected within timeout -> stop and auto-submit empty
              try {
                if (recognitionRef.current) recognitionRef.current.stop()
              } catch (e) {}
              handleStopRecording()
              if (autoSubmit) {
                onTranscriptReady('')
              }
              // clear interval and remaining
              if (silenceIntervalRef.current) {
                window.clearInterval(silenceIntervalRef.current)
                silenceIntervalRef.current = null
              }
              setSilenceRemainingSec(null)
            }, silenceTimeoutMs)
          }
        }, graceMs)
      }

      recognitionRef.current.onresult = (event: any) => {
        let interimTranscript = ''
        for (let i = event.resultIndex; i < event.results.length; i++) {
          const transcriptSegment = event.results[i][0].transcript
          if (event.results[i].isFinal) {
            setTranscript((prev) => {
              const next = prev + ' ' + transcriptSegment
              transcriptRef.current = next
              // we got speech; clear silence timer and interval
              if (silenceTimerRef.current) {
                window.clearTimeout(silenceTimerRef.current)
                silenceTimerRef.current = null
              }
              if (silenceIntervalRef.current) {
                window.clearInterval(silenceIntervalRef.current)
                silenceIntervalRef.current = null
              }
              setSilenceRemainingSec(null)
              return next
            })
          } else {
            interimTranscript += transcriptSegment
            // If we see interim (partial) speech, clear the silence timer as soon
            // as the user begins speaking so the auto-skip doesn't fire before
            // recognition finalizes.
            if (interimTranscript.trim() !== '') {
              if (silenceTimerRef.current) {
                window.clearTimeout(silenceTimerRef.current)
                silenceTimerRef.current = null
              }
              if (silenceIntervalRef.current) {
                window.clearInterval(silenceIntervalRef.current)
                silenceIntervalRef.current = null
              }
              setSilenceRemainingSec(null)
            }
          }
        }
      }

      recognitionRef.current.onerror = (event: any) => {
        console.error('Speech recognition error:', event.error)
      }

      recognitionRef.current.onend = () => {
        setIsRecording(false)
        // When recognition stops, auto-submit if enabled (use ref for latest transcript)
        if (graceTimeoutRef.current) {
          window.clearTimeout(graceTimeoutRef.current)
          graceTimeoutRef.current = null
        }
        if (silenceTimerRef.current) {
          window.clearTimeout(silenceTimerRef.current)
          silenceTimerRef.current = null
        }
        if (silenceIntervalRef.current) {
          window.clearInterval(silenceIntervalRef.current)
          silenceIntervalRef.current = null
        }
        setSilenceRemainingSec(null)
        if (autoSubmit && transcriptRef.current.trim()) {
          onTranscriptReady(transcriptRef.current.trim())
          setTranscript('')
          transcriptRef.current = ''
        }
      }
    }
  }, [])

  // Auto-start if requested
  useEffect(() => {
    if (autoStart && !disabled && !isRecording) {
      handleStartRecording()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoStart])

  const handleStartRecording = async () => {
    try {
      // Start audio recording with MediaRecorder for fallback
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const mediaRecorder = new MediaRecorder(stream)

      audioChunksRef.current = []

      mediaRecorder.ondataavailable = (event) => {
        audioChunksRef.current.push(event.data)
      }

      mediaRecorder.onstop = async () => {
        // If Web Speech API transcript is empty, we can upload to AssemblyAI
        setIsProcessing(false)
      }

      mediaRecorder.start()
      mediaRecorderRef.current = mediaRecorder

      // Start speech recognition
      if (recognitionRef.current) {
        recognitionRef.current.start()
      }

      setIsRecording(true)
      setTranscript('')
    } catch (error) {
      console.error('Error accessing microphone:', error)
    }
  }

  const handleStopRecording = () => {
    if (mediaRecorderRef.current) {
      mediaRecorderRef.current.stop()
      setIsProcessing(true)
    }

    if (recognitionRef.current) {
      recognitionRef.current.stop()
    }

    if (silenceTimerRef.current) {
      window.clearTimeout(silenceTimerRef.current)
      silenceTimerRef.current = null
    }

    setIsRecording(false)
  }

  const handleSubmitTranscript = () => {
    if (transcript.trim()) {
      onTranscriptReady(transcript.trim())
      setTranscript('')
    }
  }

  return (
    <div className="flex flex-col items-center gap-4 p-6 bg-gray-50 rounded-lg">
      <p className="text-gray-600 font-medium">Record your answer:</p>

      {isRecording && silenceRemainingSec !== null && (
        <div className="text-sm text-gray-600">Auto-skip in {silenceRemainingSec}s</div>
      )}

      {isRecording ? (
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 bg-red-500 rounded-full animate-pulse"></div>
          <span className="text-red-600 font-semibold">Recording... (Listening)</span>
        </div>
      ) : null}

      {transcript && (
        <div className="w-full p-4 bg-white rounded-lg border border-blue-300">
          <p className="text-sm text-gray-600 mb-2">Your transcript:</p>
          <p className="text-gray-800 text-lg">{transcript}</p>
        </div>
      )}

      <div className="flex gap-4">
        {!isRecording ? (
          <button
            onClick={handleStartRecording}
            disabled={disabled}
            className="px-6 py-2 bg-red-600 text-white font-semibold rounded-lg hover:bg-red-700 disabled:bg-gray-400 transition-colors"
          >
            Start Recording
          </button>
        ) : (
          <button
            onClick={handleStopRecording}
            className="px-6 py-2 bg-red-600 text-white font-semibold rounded-lg hover:bg-red-700 transition-colors"
          >
            Stop Recording
          </button>
        )}

        {transcript && !isRecording && (
          <button
            onClick={handleSubmitTranscript}
            disabled={disabled}
            className="px-6 py-2 bg-green-600 text-white font-semibold rounded-lg hover:bg-green-700 disabled:bg-gray-400 transition-colors"
          >
            Submit Answer
          </button>
        )}
      </div>
    </div>
  )
}
