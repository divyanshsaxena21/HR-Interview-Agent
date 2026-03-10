'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import FitBadge from '@/components/FitBadge'

interface InterviewSummary {
  _id: string
  candidate_name: string
  role: string
  status: string
}

interface InterviewWithEval extends InterviewSummary {
  evaluation?: {
    communication_score: number
    technical_score: number
    confidence_score: number
    fit: string
  }
  analytics?: {
    clarity_rating: number
  }
}

export default function AdminDashboardPage() {
  const router = useRouter()
  const [interviews, setInterviews] = useState<InterviewWithEval[]>([])
  const [loading, setLoading] = useState(true)
  const [authorized, setAuthorized] = useState(false)

  useEffect(() => {
    const token = localStorage.getItem('admin_token')
    if (!token) {
      router.push('/admin/login')
      return
    }
    setAuthorized(true)

    const loadInterviews = async () => {
      try {
        const response = await fetch(
          `${process.env.NEXT_PUBLIC_API_URL}/admin/interviews`,
          {
            headers: {
              'Authorization': `Bearer ${token}`,
            },
          }
        )
        if (response.ok) {
          const data = await response.json()
          setInterviews(data || [])
        } else if (response.status === 401) {
          localStorage.removeItem('admin_token')
          router.push('/admin/login')
        }
      } catch (error) {
        console.error('Error loading interviews:', error)
      } finally {
        setLoading(false)
      }
    }

    loadInterviews()
  }, [router])

  const handleLogout = () => {
    localStorage.removeItem('admin_token')
    router.push('/admin/login')
  }

  if (!authorized) {
    return null
  }

  if (loading) {
    return (
      <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center">
        <div className="text-xl text-gray-700">Loading dashboard...</div>
      </main>
    )
  }

  return (
    <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
      <div className="max-w-7xl mx-auto">
        <div className="flex justify-between items-start mb-8">
          <div>
            <h1 className="text-4xl font-bold text-gray-800 mb-2">
              Interview Dashboard
            </h1>
            <p className="text-gray-600">
              View and manage all candidate interviews
            </p>
          </div>
          <button
            onClick={handleLogout}
            className="bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700"
          >
            Logout
          </button>
        </div>

        {interviews.length === 0 ? (
          <div className="bg-white rounded-lg shadow-lg p-8 text-center">
            <p className="text-gray-600 text-lg">
              No interviews yet. Candidates can start one at the home page.
            </p>
          </div>
        ) : (
          <div className="bg-white rounded-lg shadow-lg overflow-hidden">
            <table className="w-full">
              <thead className="bg-gray-100 border-b">
                <tr>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">
                    Candidate Name
                  </th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">
                    Role
                  </th>
                  <th className="px-6 py-3 text-center text-sm font-semibold text-gray-700">
                    Communication
                  </th>
                  <th className="px-6 py-3 text-center text-sm font-semibold text-gray-700">
                    Technical
                  </th>
                  <th className="px-6 py-3 text-center text-sm font-semibold text-gray-700">
                    Confidence
                  </th>
                  <th className="px-6 py-3 text-center text-sm font-semibold text-gray-700">
                    Fit
                  </th>
                  <th className="px-6 py-3 text-center text-sm font-semibold text-gray-700">
                    Status
                  </th>
                  <th className="px-6 py-3 text-center text-sm font-semibold text-gray-700">
                    Action
                  </th>
                </tr>
              </thead>
              <tbody>
                {interviews.map((interview) => (
                  <tr key={interview._id} className="border-b hover:bg-gray-50">
                    <td className="px-6 py-4 text-gray-800">
                      {interview.candidate_name}
                    </td>
                    <td className="px-6 py-4 text-gray-600">{interview.role}</td>
                    <td className="px-6 py-4 text-center">
                      {interview.evaluation?.communication_score ? (
                        <span className="font-semibold text-blue-600">
                          {interview.evaluation.communication_score}/10
                        </span>
                      ) : (
                        <span className="text-gray-400">-</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-center">
                      {interview.evaluation?.technical_score ? (
                        <span className="font-semibold text-green-600">
                          {interview.evaluation.technical_score}/10
                        </span>
                      ) : (
                        <span className="text-gray-400">-</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-center">
                      {interview.evaluation?.confidence_score ? (
                        <span className="font-semibold text-purple-600">
                          {interview.evaluation.confidence_score}/10
                        </span>
                      ) : (
                        <span className="text-gray-400">-</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-center">
                      {interview.evaluation?.fit ? (
                        <FitBadge fit={interview.evaluation.fit} />
                      ) : (
                        <span className="text-gray-400">-</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-center">
                      <span
                        className={`px-3 py-1 rounded-full text-sm font-semibold ${
                          interview.status === 'completed'
                            ? 'bg-green-100 text-green-800'
                            : interview.status === 'in_progress'
                            ? 'bg-blue-100 text-blue-800'
                            : 'bg-yellow-100 text-yellow-800'
                        }`}
                      >
                        {interview.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-center">
                      <a
                        href={`/interview/${interview._id}`}
                        className="text-blue-600 hover:text-blue-800 font-semibold"
                      >
                        View
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </main>
  )
}
