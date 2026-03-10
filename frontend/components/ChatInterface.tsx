'use client'

import { useEffect, useRef, useState } from 'react'
import { io, Socket } from 'socket.io-client'
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
  const [messages, setMessages] = useState<Message[]>([])
  const [inputValue, setInputValue] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [showInfoForm, setShowInfoForm] = useState(false)
  const [github, setGithub] = useState('')
  const [linkedin, setLinkedin] = useState('')
  const [portfolio, setPortfolio] = useState('')
  const [infoSaving, setInfoSaving] = useState(false)
  const socketRef = useRef<Socket | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const socketUrl = process.env.NEXT_PUBLIC_SOCKET_URL || 'http://localhost:8080'
    socketRef.current = io(socketUrl, {
      query: { interviewId, candidateName },
    })

    socketRef.current.on('ai_message', (message: Message) => {
      setMessages(prev => [...prev, message])
      setIsLoading(false)
    })

    socketRef.current.on('error', (error: string) => {
      console.error('Socket error:', error)
      setIsLoading(false)
    })

    return () => {
      if (socketRef.current) {
        socketRef.current.disconnect()
      }
    }
  }, [interviewId, candidateName])

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

    const userMessage: Message = {
      role: 'candidate',
      content: inputValue,
      timestamp: Date.now(),
    }

    setMessages(prev => [...prev, userMessage])
    setInputValue('')
    setIsLoading(true)

    if (socketRef.current) {
      socketRef.current.emit('candidate_message', {
        interviewId,
        message: inputValue,
      })
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

  return (
    <div className="flex flex-col h-screen bg-gray-50">
      <div className="bg-white border-b p-4 shadow-sm">
        <h1 className="text-2xl font-bold text-gray-800">Interview Chat</h1>
        <p className="text-gray-600">Candidate: {candidateName}</p>
        <div className="mt-2">
          <button onClick={()=>setShowInfoForm(s=>!s)} className="text-sm text-blue-600 hover:underline">{showInfoForm ? 'Cancel' : 'Update candidate info'}</button>
        </div>
        {showInfoForm && (
          <div className="mt-2 grid grid-cols-1 gap-2">
            <input value={github} onChange={e=>setGithub(e.target.value)} placeholder="GitHub URL" className="px-3 py-2 border rounded" />
            <input value={linkedin} onChange={e=>setLinkedin(e.target.value)} placeholder="LinkedIn URL" className="px-3 py-2 border rounded" />
            <input value={portfolio} onChange={e=>setPortfolio(e.target.value)} placeholder="Portfolio URL" className="px-3 py-2 border rounded" />
            <div>
              <button onClick={handleSaveInfo} disabled={infoSaving} className="px-3 py-1 bg-green-500 text-white rounded">{infoSaving ? 'Saving...' : 'Save'}</button>
            </div>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.length === 0 && (
          <div className="flex items-center justify-center h-full text-gray-500">
            <p>Start by typing your response...</p>
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
            onKeyPress={e => e.key === 'Enter' && handleSendMessage()}
            placeholder="Type your response..."
            disabled={isLoading}
            className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
          />
          <button
            onClick={handleSendMessage}
            disabled={isLoading || !inputValue.trim()}
            className="px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-400 font-medium"
          >
            Send
          </button>
        </div>
      </div>
    </div>
  )
}
