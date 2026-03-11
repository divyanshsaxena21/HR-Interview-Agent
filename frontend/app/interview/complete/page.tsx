'use client'

import { useRouter } from 'next/navigation'
import { useState, useEffect } from 'react'

export default function InterviewCompletePage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)

  const handleReturnHome = () => {
    setIsLoading(true)
    router.push('/')
  }

  useEffect(() => {
    // Auto-redirect after 10 seconds
    const timer = setTimeout(() => {
      router.push('/')
    }, 10000)

    return () => clearTimeout(timer)
  }, [router])

  return (
    <div className="min-h-screen bg-gradient-to-br from-green-50 to-blue-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-2xl p-12 w-full max-w-md text-center">
        <div className="mb-6">
          <div className="flex justify-center mb-4">
            <div className="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center">
              <svg
                className="w-10 h-10 text-green-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </div>
          </div>
          <h1 className="text-3xl font-bold text-gray-800 mb-2">
            Interview Complete!
          </h1>
          <p className="text-gray-600 mb-6">
            Thank you for taking the time to interview with us.
          </p>
        </div>

        <div className="bg-blue-50 rounded-lg p-6 mb-6 border border-blue-200">
          <p className="text-sm text-gray-700 leading-relaxed">
            We've received your responses and documents. Our recruiting team will review your interview and get back to you within the next few days.
          </p>
        </div>

        <div className="space-y-3 text-sm text-gray-600 mb-8">
          <div className="flex items-start">
            <svg className="w-5 h-5 text-green-500 mr-2 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
            <span>Your responses have been saved</span>
          </div>
          <div className="flex items-start">
            <svg className="w-5 h-5 text-green-500 mr-2 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
            <span>Documents have been uploaded securely</span>
          </div>
          <div className="flex items-start">
            <svg className="w-5 h-5 text-green-500 mr-2 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
            <span>A confirmation email has been sent</span>
          </div>
        </div>

        <div className="text-xs text-gray-500 mb-6">
          Redirecting to home in a few seconds...
        </div>

        <button
          onClick={handleReturnHome}
          disabled={isLoading}
          className="w-full py-3 px-4 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-400 font-medium transition"
        >
          {isLoading ? 'Redirecting...' : 'Return Home'}
        </button>
      </div>
    </div>
  )
}
