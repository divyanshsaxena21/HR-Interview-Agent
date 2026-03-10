'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import CandidateForm from '@/components/CandidateForm'

export default function Home() {
  const router = useRouter()

  const handleStartInterview = async (data: {
    name: string
    email: string
    role: string
  }) => {
    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/interview/start`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(data),
        }
      )

      if (response.ok) {
        const result = await response.json()
        router.push(`/interview/${result.interview_id}`)
      }
    } catch (error) {
      console.error('Error starting interview:', error)
    }
  }

  return (
    <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-xl p-8 w-full max-w-md">
        <h1 className="text-4xl font-bold text-gray-800 mb-2 text-center">
          AI Recruiter
        </h1>
        <p className="text-gray-600 text-center mb-8">
          Automated screening interviews powered by AI
        </p>
        <CandidateForm onSubmit={handleStartInterview} />
        
        <div className="mt-8 pt-8 border-t border-gray-200 text-center">
          <p className="text-gray-600 text-sm mb-3">Are you an admin?</p>
          <a
            href="/admin/login"
            className="inline-block bg-gray-800 text-white px-4 py-2 rounded-lg hover:bg-gray-900 font-semibold"
          >
            Admin Login
          </a>
        </div>
      </div>
    </main>
  )
}
