'use client'

import { useEffect, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import axios from 'axios'

interface Message {
  role: 'candidate' | 'ai'
  content: string
  timestamp: number
}

interface ChatInterfaceProps {
  interviewId: string
  candidateName: string
}

export default function ChatInterface({ interviewId, candidateName }: ChatInterfaceProps) {
  const router = useRouter()
  const [messages, setMessages] = useState<Message[]>([])
  const [inputValue, setInputValue] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [showInfoForm, setShowInfoForm] = useState(false)
  const [github, setGithub] = useState('')
  const [linkedin, setLinkedin] = useState('')
  const [portfolio, setPortfolio] = useState('')
  const [infoSaving, setInfoSaving] = useState(false)
  const [interviewFinished, setInterviewFinished] = useState(false)
  const [finishingInterview, setFinishingInterview] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    // Only connect if we don't already have an active connection
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      console.log('[WS] Already connected, skipping reconnect')
      return
    }

    const wsUrl = process.env.NEXT_PUBLIC_SOCKET_URL || 'http://localhost:8080'
    const wsProtocol = wsUrl.startsWith('https') ? 'wss' : 'ws'
    const wsPath = `${wsProtocol}://${wsUrl.replace(/^https?:\/\//, '')}/ws/${interviewId}/${candidateName}`
    
    console.log('[WS] Connecting to:', wsPath)
    
    const ws = new WebSocket(wsPath)
    let isMounted = true
    
    ws.onopen = () => {
      if (isMounted) {
        console.log('[WS] ✓ Connected')
      }
    }
    
    ws.onmessage = (event) => {
      if (!isMounted) return
      try {
        const data = JSON.parse(event.data)
        console.log('[WS] ✓ Received:', data.type)
        
        if (data.type === 'ai_message' || data.type === 'interview_ended') {
          setMessages(prev => [...prev, {
            role: 'ai',
            content: data.content,
            timestamp: Date.now()
          }])
          setIsLoading(false)
          
          // If interview is ended, auto-redirect after showing the message
          if (data.type === 'interview_ended') {
            console.log('[Interview] Completed - redirecting to completion page')
            setTimeout(() => {
              window.location.href = '/interview/complete'
            }, 2000) // Give 2 seconds to read the final message
          }
        } else if (data.type === 'error') {
          console.error('[WS] Server error:', data.content)
          setIsLoading(false)
        }
      } catch (err) {
        console.error('[WS] ✗ Parse error:', err)
      }
    }
    
    ws.onerror = (error) => {
      if (isMounted) {
        console.error('[WS] ✗ Connection error:', error)
      }
    }
    
    ws.onclose = () => {
      if (isMounted) {
        console.log('[WS] Disconnected')
      }
    }
    
    wsRef.current = ws
    
    return () => {
      isMounted = false
      // Don't close immediately - let the connection persist across hot reloads
      // Only close when component actually unmounts (interviewId changes)
    }
  }, [interviewId, candidateName])

  // Cleanup on actual unmount
  useEffect(() => {
    return () => {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        console.log('[WS] Component unmounting, closing connection')
        wsRef.current.close()
      }
    }
  }, [])

  // Fetch existing messages and request initial AI question if none
  useEffect(() => {
    const load = async () => {
      try {
        const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
        const res = await axios.get(`${apiUrl}/interview/${interviewId}`)
        const msgs = (res.data.messages || []).map((m: any) => ({ role: m.role, content: m.content, timestamp: m.timestamp }))
        setMessages(msgs)
        if (msgs.length === 0) {
          const r = await axios.post(`${apiUrl}/interview/${interviewId}/ai-start`)
          if (r.data?.question) {
            setMessages([{ role: 'ai', content: r.data.question, timestamp: Date.now() }, ...msgs])
          }
        }
      } catch (err) {
        console.error('Failed to load interview messages or init question', err)
      }
    }
    load()
  }, [interviewId])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSendMessage = () => {
    if (!inputValue.trim() || isLoading) return

    console.log('[WS] Sending message:', inputValue)

    const userMessage: Message = {
      role: 'candidate',
      content: inputValue,
      timestamp: Date.now(),
    }

    setMessages(prev => [...prev, userMessage])
    setInputValue('')
    setIsLoading(true)

    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'candidate_message',
        content: inputValue,
      }))
    } else {
      console.error('[WS] ✗ WebSocket not connected')
      setIsLoading(false)
    }
  }

  const handleSaveInfo = async () => {
    setInfoSaving(true)
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      await axios.put(`${apiUrl}/interview/${interviewId}/info`, {
        github,
        linkedin,
        portfolio,
      })
      setShowInfoForm(false)
    } catch (err) {
      console.error('Failed to save candidate info', err)
    } finally {
      setInfoSaving(false)
    }
  }

  const handleFinishInterview = async () => {
    if (!window.confirm('Are you sure you want to finish the interview? You can no longer send messages after this.')) {
      return
    }

    setFinishingInterview(true)
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      await axios.post(`${apiUrl}/interview/${interviewId}/end`)
      
      // Add thank you message to chat
      setMessages(prev => [...prev, {
        role: 'ai',
        content: 'Thank you for your time! We appreciate your participation in this interview. Our team will review your responses and get back to you soon.',
        timestamp: Date.now()
      }])
      
      setInterviewFinished(true)
      if (wsRef.current) {
        wsRef.current.close()
      }

      // Redirect to completion page after 4 seconds so candidate can see the thank you message
      setTimeout(() => {
        router.push('/interview/complete')
      }, 4000)
    } catch (err) {
      console.error('Failed to finish interview', err)
      alert('Failed to finish interview. Please try again.')
    } finally {
      setFinishingInterview(false)
    }
  }

  return (
    <div className="flex flex-col h-screen bg-gray-50">
      <div className="bg-white border-b p-4 shadow-sm">
        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-2xl font-bold text-gray-800">Interview Chat</h1>
            <p className="text-gray-600">Candidate: {candidateName}</p>
            {interviewFinished && (
              <p className="text-sm font-semibold text-green-600 mt-2">✓ Interview completed</p>
            )}
          </div>
          <div className="flex flex-col gap-2">
            <button 
              onClick={() => setShowInfoForm(s => !s)} 
              disabled={interviewFinished}
              className="text-sm text-blue-600 hover:underline disabled:text-gray-400 disabled:cursor-not-allowed"
            >
              {showInfoForm ? 'Cancel' : 'Update candidate info'}
            </button>
            <button
              onClick={handleFinishInterview}
              disabled={interviewFinished || finishingInterview || isLoading}
              className="text-sm px-3 py-1 bg-red-500 text-white rounded hover:bg-red-600 disabled:bg-gray-400 disabled:cursor-not-allowed font-medium"
            >
              {finishingInterview ? 'Finishing...' : 'Finish Interview'}
            </button>
          </div>
        </div>
        {showInfoForm && (
          <div className="mt-4 grid grid-cols-1 gap-2">
            <input value={github} onChange={e=>setGithub(e.target.value)} placeholder="GitHub URL" className="px-3 py-2 border rounded" />
            <input value={linkedin} onChange={e=>setLinkedin(e.target.value)} placeholder="LinkedIn URL" className="px-3 py-2 border rounded" />
            <input value={portfolio} onChange={e=>setPortfolio(e.target.value)} placeholder="Portfolio URL" className="px-3 py-2 border rounded" />
            <div>
              <button onClick={handleSaveInfo} disabled={infoSaving} className="px-3 py-1 bg-green-500 text-white rounded hover:bg-green-600 disabled:bg-gray-400">{infoSaving ? 'Saving...' : 'Save'}</button>
            </div>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.length === 0 && !interviewFinished && (
          <div className="flex items-center justify-center h-full text-gray-500">
            <p>Start by typing your response...</p>
          </div>
        )}
        {interviewFinished && messages.length > 0 && (
          <div className="flex items-center justify-center p-4 bg-green-50 rounded-lg border border-green-200 text-green-700 mb-4">
            <p>✓ Interview completed. Thank you for your time!</p>
          </div>
        )}
        {messages.map((msg, idx) => (
          <div
            key={idx}
            className={`flex ${msg.role === 'candidate' ? 'justify-end' : 'justify-start'}`}
          >
            <div
              className={`max-w-xs lg:max-w-md xl:max-w-lg px-4 py-3 rounded-lg ${
                msg.role === 'candidate'
                  ? 'bg-blue-500 text-white rounded-br-none'
                  : 'bg-gray-200 text-gray-900 rounded-bl-none'
              }`}
            >
              <p className="text-sm">{msg.content}</p>
            </div>
          </div>
        ))}
        {isLoading && (
          <div className="flex justify-start">
            <div className="bg-gray-200 text-gray-900 px-4 py-3 rounded-lg rounded-bl-none">
              <div className="flex space-x-2">
                <div className="w-2 h-2 bg-gray-600 rounded-full animate-bounce"></div>
                <div className="w-2 h-2 bg-gray-600 rounded-full animate-bounce delay-100"></div>
                <div className="w-2 h-2 bg-gray-600 rounded-full animate-bounce delay-200"></div>
              </div>
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <div className="bg-white border-t p-4">
        <div className="flex gap-2">
          <input
            type="text"
            value={inputValue}
            onChange={e => setInputValue(e.target.value)}
            onKeyPress={e => e.key === 'Enter' && !interviewFinished && handleSendMessage()}
            placeholder={interviewFinished ? "Interview completed" : "Type your response..."}
            disabled={isLoading || interviewFinished}
            className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
          />
          <button
            onClick={handleSendMessage}
            disabled={isLoading || !inputValue.trim() || interviewFinished}
            className="px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-400 disabled:cursor-not-allowed font-medium"
          >
            Send
          </button>
        </div>
      </div>
    </div>
  )
}
