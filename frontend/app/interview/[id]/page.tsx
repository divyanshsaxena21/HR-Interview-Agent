'use client'

import { useParams } from 'next/navigation'
import { useEffect, useState } from 'react'
import ChatInterface from '@/components/ChatInterface'
import axios from 'axios'

interface InterviewData {
  id: string
  candidate_name: string
  email: string
  role: string
  github?: string
  linkedin?: string
  portfolio?: string
  messages: any[]
  status: string
  rejected: boolean
}

export default function InterviewDetailPage() {
  const params = useParams()
  const id = params.id as string
  const [interview, setInterview] = useState<InterviewData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const fetchInterview = async () => {
      try {
        const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
        const response = await axios.get(`${apiUrl}/interview/${id}`)
        setInterview(response.data)
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to load interview')
      } finally {
        setLoading(false)
      }
    }

    fetchInterview()
  }, [id])

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mb-4"></div>
          <p className="text-gray-600">Loading interview...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <p className="text-red-600 mb-4">{error}</p>
          <a href="/" className="text-blue-500 hover:underline">
            Return to home
          </a>
        </div>
      </div>
    )
  }

  if (!interview) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <p className="text-gray-600">Interview not found</p>
      </div>
    )
  }

  return <ChatInterface interviewId={id} candidateName={interview.candidate_name} />
}
