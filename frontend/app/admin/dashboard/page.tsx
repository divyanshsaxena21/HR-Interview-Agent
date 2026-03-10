'use client'

import React, { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import axios from 'axios'
import FitBadge from '../../../components/FitBadge'

// Ensure axios requests target the backend API when running in dev
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
if (API_BASE) axios.defaults.baseURL = API_BASE

interface Interview {
  id: string
  candidate_name: string
  email: string
  role: string
  github?: string
  linkedin?: string
  portfolio?: string
  messages?: any[]
  status?: string
  rejected?: boolean
  rejection_reason?: string
  created_at?: string
  evaluation?: any
}

interface HRQuestion {
  _id?: string
  category: string
  question: string
  tags: string[]
  is_dealbreaker: boolean
  active: boolean
}

type TabType = 'interviews' | 'hr-memory'

export default function AdminDashboardPage() {
  const router = useRouter()
  const [activeTab, setActiveTab] = useState<TabType>('interviews')
  const [interviews, setInterviews] = useState<Interview[]>([])
  const [questions, setQuestions] = useState<HRQuestion[]>([])
  const [loading, setLoading] = useState(false)
  const [showNewQuestionForm, setShowNewQuestionForm] = useState(false)
  const [newQuestion, setNewQuestion] = useState<HRQuestion>({
    category: '',
    question: '',
    tags: [],
    is_dealbreaker: false,
    active: true,
  })

  useEffect(() => {
    fetchData()
  }, [activeTab])

  async function fetchData() {
    setLoading(true)
    try {
      if (activeTab === 'interviews') {
        const token = localStorage.getItem('token')
        if (!token) return router.push('/admin/login')
        const res = await axios.get('/admin/interviews', { headers: { Authorization: `Bearer ${token}` } })
        setInterviews(res.data || [])
      } else {
        const token = localStorage.getItem('token')
        if (!token) return router.push('/admin/login')
        const res = await axios.get('/admin/questions', { headers: { Authorization: `Bearer ${token}` } })
        setQuestions(res.data || [])
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  async function handleAddQuestion() {
    try {
      setLoading(true)
      const token = localStorage.getItem('token')
      if (!token) return router.push('/admin/login')
      const payload = { ...newQuestion, tags: Array.isArray(newQuestion.tags) ? newQuestion.tags : String(newQuestion.tags).split(',').map(t=>t.trim()).filter(Boolean) }
      await axios.post('/admin/questions', payload, { headers: { Authorization: `Bearer ${token}` } })
      setNewQuestion({ category: '', question: '', tags: [], is_dealbreaker: false, active: true })
      setShowNewQuestionForm(false)
      fetchData()
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  async function handleDeleteQuestion(id?: string) {
    if (!id) return
    if (!confirm('Delete this question?')) return
    try {
      setLoading(true)
      const token = localStorage.getItem('token')
      if (!token) return router.push('/admin/login')
      await axios.delete(`/admin/questions/${id}`, { headers: { Authorization: `Bearer ${token}` } })
      fetchData()
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  function handleLogout() {
    // placeholder logout; adapt to your auth flow
    try {
      localStorage.removeItem('token')
      localStorage.removeItem('adminToken')
      localStorage.removeItem('adminName')
    } finally {
      router.push('/admin/login')
    }
  }

  return (
    <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
      <div className="max-w-7xl mx-auto">
        <div className="flex justify-between items-start mb-8">
          <div>
            <h1 className="text-4xl font-bold text-gray-800 mb-2">Interview Dashboard</h1>
            <p className="text-gray-600">View and manage all candidate interviews</p>
          </div>
          <button
            onClick={handleLogout}
            className="bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700"
          >
            Logout
          </button>
        </div>

        <div className="bg-white rounded-lg shadow p-4 mb-6">
          <div className="flex border-b bg-white">
            <button
              onClick={() => setActiveTab('interviews')}
              className={`px-6 py-3 font-medium ${
                activeTab === 'interviews'
                  ? 'text-blue-600 border-b-2 border-blue-600'
                  : 'text-gray-600 hover:text-gray-800'
              }`}
            >
              Interviews
            </button>
            <button
              onClick={() => setActiveTab('hr-memory')}
              className={`px-6 py-3 font-medium ${
                activeTab === 'hr-memory'
                  ? 'text-blue-600 border-b-2 border-blue-600'
                  : 'text-gray-600 hover:text-gray-800'
              }`}
            >
              Question Library
            </button>
          </div>

          <div className="p-6">
            {activeTab === 'interviews' && (
              <div>
                <h2 className="text-xl font-bold mb-4">Interview Results</h2>
                {loading ? (
                  <p>Loading...</p>
                ) : interviews.length === 0 ? (
                  <p className="text-gray-600">No interviews yet</p>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full bg-white rounded-lg shadow">
                      <thead className="bg-gray-50 border-b">
                        <tr>
                          <th className="px-6 py-3 text-left text-sm font-medium text-gray-700">Candidate</th>
                          <th className="px-6 py-3 text-left text-sm font-medium text-gray-700">Role</th>
                          <th className="px-6 py-3 text-left text-sm font-medium text-gray-700">Status</th>
                          <th className="px-6 py-3 text-left text-sm font-medium text-gray-700">Rejected</th>
                          <th className="px-6 py-3 text-left text-sm font-medium text-gray-700">Links</th>
                        </tr>
                      </thead>
                      <tbody>
                        {interviews.map((interview) => (
                          <tr key={interview.id} className="border-b hover:bg-gray-50">
                            <td className="px-6 py-3">
                              <div>
                                <p className="font-medium">{interview.candidate_name}</p>
                                <p className="text-sm text-gray-600">{interview.email}</p>
                              </div>
                            </td>
                            <td className="px-6 py-3">{interview.role}</td>
                            <td className="px-6 py-3">
                              <span className="px-2 py-1 bg-blue-100 text-blue-800 rounded text-sm">{interview.status}</span>
                            </td>
                            <td className="px-6 py-3">{interview.rejected ? <span className="px-2 py-1 bg-red-100 text-red-800 rounded text-sm">Rejected</span> : <span className="text-gray-600">No</span>}</td>
                            <td className="px-6 py-3 text-sm">
                              {interview.github && (
                                <a href={interview.github} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline mr-3">GitHub</a>
                              )}
                              {interview.linkedin && (
                                <a href={interview.linkedin} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">LinkedIn</a>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'hr-memory' && (
              <div>
                <div className="flex justify-between items-center mb-4">
                  <h2 className="text-xl font-bold">Interview Question Library</h2>
                  <button
                    onClick={() => setShowNewQuestionForm(!showNewQuestionForm)}
                    className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600"
                  >
                    {showNewQuestionForm ? 'Cancel' : 'Add Question'}
                  </button>
                </div>

                {showNewQuestionForm && (
                  <div className="bg-white p-6 rounded-lg shadow mb-6">
                    <div className="space-y-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Category</label>
                        <input type="text" value={newQuestion.category} onChange={e=>setNewQuestion({...newQuestion, category: e.target.value})} placeholder="e.g., general, technical" className="w-full px-4 py-2 border border-gray-300 rounded-lg" />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Question</label>
                        <textarea value={newQuestion.question} onChange={e=>setNewQuestion({...newQuestion, question: e.target.value})} rows={3} className="w-full px-4 py-2 border border-gray-300 rounded-lg" />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Tags (comma-separated)</label>
                        <input type="text" value={Array.isArray(newQuestion.tags) ? newQuestion.tags.join(',') : String(newQuestion.tags)} onChange={e=>setNewQuestion({...newQuestion, tags: String(e.target.value).split(',').map(t=>t.trim()).filter(Boolean)})} className="w-full px-4 py-2 border border-gray-300 rounded-lg" />
                      </div>
                      <div className="flex items-center gap-4">
                        <label className="flex items-center gap-2"><input type="checkbox" checked={newQuestion.is_dealbreaker} onChange={e=>setNewQuestion({...newQuestion, is_dealbreaker: e.target.checked})} className="rounded" /> <span className="text-sm font-medium text-gray-700">Dealbreaker Question</span></label>
                        <label className="flex items-center gap-2"><input type="checkbox" checked={newQuestion.active} onChange={e=>setNewQuestion({...newQuestion, active: e.target.checked})} className="rounded" /> <span className="text-sm font-medium text-gray-700">Active</span></label>
                      </div>
                      <button onClick={handleAddQuestion} className="w-full px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600">Add Question</button>
                    </div>
                  </div>
                )}

                {loading ? (
                  <p>Loading...</p>
                ) : questions.length === 0 ? (
                  <p className="text-gray-600">No questions yet</p>
                ) : (
                  <div className="space-y-4">
                    {questions.map(q => (
                      <div key={q._id} className="bg-white p-4 rounded-lg shadow">
                        <div className="flex justify-between items-start">
                          <div className="flex-1">
                            <p className="font-medium text-gray-800">{q.question}</p>
                            <div className="mt-2 flex gap-2">
                              <span className="text-xs bg-gray-100 text-gray-700 px-2 py-1 rounded">{q.category}</span>
                              {q.is_dealbreaker && <span className="text-xs bg-red-100 text-red-700 px-2 py-1 rounded">Dealbreaker</span>}
                              {!q.active && <span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-1 rounded">Inactive</span>}
                            </div>
                            {q.tags && q.tags.length > 0 && (
                              <div className="mt-2">{q.tags.map(tag => <span key={tag} className="text-xs text-gray-600 mr-2">#{tag}</span>)}</div>
                            )}
                          </div>
                          <button onClick={()=>handleDeleteQuestion(q._id)} className="ml-4 px-3 py-1 bg-red-500 text-white rounded hover:bg-red-600 text-sm">Delete</button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Fallback content if no interviews */}
        {interviews.length === 0 && activeTab === 'interviews' && (
          <div className="bg-white rounded-lg shadow-lg p-8 text-center">
            <p className="text-gray-600 text-lg">No interviews yet. Candidates can start one at the home page.</p>
          </div>
        )}
      </div>
    </main>
  )
}
