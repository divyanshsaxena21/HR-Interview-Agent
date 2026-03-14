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
  documents?: Array<{
    file_name: string
    content_type: string
    data: string
    uploaded_at: number
  }>
  messages?: Array<{
    role: string
    content: string
  }>
  message_count: number
  status?: string
  rejected?: boolean
  rejection_reason?: string
  created_at?: string
  evaluation_id?: string
  evaluation?: {
    id?: string
    communication_score?: number
    technical_score?: number
    confidence_score?: number
    problem_solving_score?: number
    fit?: string
    summary?: string
    strengths?: string[]
    weaknesses?: string[]
  }
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
  const [expandedId, setExpandedId] = useState<string | null>(null)
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

  async function handleDeleteInterview(id: string, candidateName: string) {
    if (!confirm(`Delete interview for ${candidateName}? This cannot be undone.`)) return
    try {
      setLoading(true)
      const token = localStorage.getItem('token')
      if (!token) return router.push('/admin/login')
      await axios.delete(`/admin/interviews/${id}`, { headers: { Authorization: `Bearer ${token}` } })
      alert('Interview deleted successfully')
      fetchData()
    } catch (err) {
      console.error(err)
      alert('Failed to delete interview')
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
          <div className="flex gap-3">
            <button
              onClick={() => router.push('/admin/candidates')}
              className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"
            >
              Manage Candidates
            </button>
            <button
              onClick={handleLogout}
              className="bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700"
            >
              Logout
            </button>
          </div>
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
                  <div className="bg-white rounded-lg shadow overflow-hidden">
                    <table className="w-full">
                      {/* Table Header */}
                      <thead>
                        <tr className="bg-gray-200 border-b-2 border-gray-300">
                          <th className="px-6 py-3 text-left text-xs font-bold text-gray-700 w-1"></th>
                          <th className="px-4 py-3 text-left text-xs font-bold text-gray-700">Candidate</th>
                          <th className="px-4 py-3 text-left text-xs font-bold text-gray-700">Role</th>
                          <th className="px-4 py-3 text-left text-xs font-bold text-gray-700">Status</th>
                          <th className="px-4 py-3 text-center text-xs font-bold text-gray-700">Messages</th>
                          <th className="px-4 py-3 text-left text-xs font-bold text-gray-700">Interview Summary</th>
                          <th className="px-4 py-3 text-center text-xs font-bold text-gray-700">Fit</th>
                        </tr>
                      </thead>

                      {/* Table Body */}
                      <tbody>
                        {interviews.map((interview) => (
                          <React.Fragment key={interview.id}>
                            <tr className="border-b hover:bg-gray-50 cursor-pointer" onClick={() => setExpandedId(expandedId === interview.id ? null : interview.id)}>
                              <td className="px-6 py-3 w-1">
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    setExpandedId(expandedId === interview.id ? null : interview.id)
                                  }}
                                  className="text-xl text-gray-600 hover:text-gray-800"
                                >
                                  {expandedId === interview.id ? '▼' : '▶'}
                                </button>
                              </td>
                              <td className="px-4 py-3">
                                <div>
                                  <p className="font-medium text-sm">{interview.candidate_name}</p>
                                  <p className="text-xs text-gray-500">{interview.email}</p>
                                </div>
                              </td>
                              <td className="px-4 py-3 text-sm">{interview.role}</td>
                              <td className="px-4 py-3">
                                <div className="flex items-center gap-2">
                                  <span className={`px-2 py-1 rounded text-xs font-medium whitespace-nowrap ${
                                    interview.status === 'completed' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                                  }`}>
                                    {interview.status}
                                  </span>
                                  {interview.rejected && (
                                    <span className="px-2 py-1 rounded text-xs font-medium bg-red-100 text-red-800 whitespace-nowrap">Rejected</span>
                                  )}
                                </div>
                              </td>
                              <td className="px-4 py-3 text-center text-sm font-medium text-gray-800">{interview.message_count || 0}</td>
                              <td className="px-4 py-3 text-left text-sm">
                                {interview.evaluation?.summary ? (
                                  <p className="text-gray-700 line-clamp-2">{interview.evaluation.summary}</p>
                                ) : (
                                  <span className="text-gray-400 text-xs">No summary available</span>
                                )}
                              </td>
                              <td className="px-4 py-3 text-center">
                                {interview.evaluation?.fit && (
                                  <span className={`px-2 py-1 rounded text-xs font-medium whitespace-nowrap ${
                                    interview.evaluation.fit === 'strong_yes' ? 'bg-green-100 text-green-800' :
                                    interview.evaluation.fit === 'yes' ? 'bg-blue-100 text-blue-800' :
                                    interview.evaluation.fit === 'maybe' ? 'bg-yellow-100 text-yellow-800' :
                                    'bg-red-100 text-red-800'
                                  }`}>
                                    {interview.evaluation.fit.replace('_', ' ')}
                                  </span>
                                )}
                              </td>
                              </tr>

                              {/* Expanded Details Row */}
                              {expandedId === interview.id && (
                                <tr className="border-t">
                                  <td colSpan={10} className="px-6 py-4 bg-gray-50">
                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                                      {/* Candidate Details */}
                                      <div>
                                        <h4 className="font-bold text-gray-800 mb-3">📋 Candidate Details</h4>
                                        <div className="space-y-2 text-sm">
                                          <div>
                                            <p className="text-gray-600">Name:</p>
                                            <p className="font-medium">{interview.candidate_name}</p>
                                          </div>
                                          <div>
                                            <p className="text-gray-600">Email:</p>
                                            <p className="font-medium">{interview.email}</p>
                                          </div>
                                          <div>
                                            <p className="text-gray-600">Role Applied:</p>
                                            <p className="font-medium">{interview.role}</p>
                                          </div>
                                          <div>
                                            <p className="text-gray-600">Created:</p>
                                            <p className="font-medium">{new Date(interview.created_at || '').toLocaleDateString()}</p>
                                          </div>
                                        </div>
                                      </div>

                                      {/* Links & Documents */}
                                      <div>
                                        <h4 className="font-bold text-gray-800 mb-3">🔗 Links & Documents</h4>
                                        <div className="space-y-3">
                                          {interview.github && (
                                            <div className="flex items-center gap-2">
                                              <span className="text-sm text-gray-600 font-medium">GitHub:</span>
                                              <a
                                                href={interview.github}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                className="text-blue-600 hover:underline text-sm break-all"
                                              >
                                                {interview.github}
                                              </a>
                                            </div>
                                          )}
                                          {interview.linkedin && (
                                            <div className="flex items-center gap-2">
                                              <span className="text-sm text-gray-600 font-medium">LinkedIn:</span>
                                              <a
                                                href={interview.linkedin}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                className="text-blue-600 hover:underline text-sm break-all"
                                              >
                                                {interview.linkedin}
                                              </a>
                                            </div>
                                          )}
                                          {interview.portfolio && (
                                            <div className="flex items-center gap-2">
                                              <span className="text-sm text-gray-600 font-medium">Portfolio:</span>
                                              <a
                                                href={interview.portfolio}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                className="text-blue-600 hover:underline text-sm break-all"
                                              >
                                                {interview.portfolio}
                                              </a>
                                            </div>
                                          )}
                                          {!interview.github && !interview.linkedin && !interview.portfolio && (
                                            <p className="text-gray-500 text-sm">No links provided</p>
                                          )}
                                        </div>

                                        {/* Uploaded Documents */}
                                        {interview.documents && interview.documents.length > 0 && (
                                          <div className="mt-4 pt-4 border-t">
                                            <h5 className="font-semibold text-gray-800 mb-3 text-sm">📄 Uploaded Documents</h5>
                                            <div className="space-y-2">
                                              {interview.documents.map((doc, idx) => (
                                                <div key={idx} className="flex items-center justify-between bg-white p-2 rounded border border-gray-200">
                                                  <div className="flex-1">
                                                    <p className="text-sm font-medium text-gray-800">{doc.file_name}</p>
                                                    <p className="text-xs text-gray-500">
                                                      {new Date(doc.uploaded_at * 1000).toLocaleDateString()} · {doc.content_type}
                                                    </p>
                                                  </div>
                                                  <button
                                                    onClick={() => {
                                                      const link = document.createElement('a')
                                                      const blob = new Blob([Buffer.from(doc.data, 'base64')], { type: doc.content_type })
                                                      link.href = URL.createObjectURL(blob)
                                                      link.download = doc.file_name
                                                      link.click()
                                                    }}
                                                    className="ml-2 px-3 py-1 bg-blue-500 text-white text-xs rounded hover:bg-blue-600"
                                                  >
                                                    Download
                                                  </button>
                                                </div>
                                              ))}
                                            </div>
                                          </div>
                                        )}
                                      </div>

                                      {/* Rejection Info */}
                                      {interview.rejected && (
                                        <div className="md:col-span-2 bg-red-50 border border-red-200 rounded-lg p-4">
                                          <h4 className="font-bold text-red-800 mb-2">❌ Rejection Reason</h4>
                                          <p className="text-red-700 text-sm">{interview.rejection_reason || 'Dealbreaker question failed'}</p>
                                        </div>
                                      )}

                                      {/* Interview Transcript */}
                                      <div className="md:col-span-2 bg-gray-50 border border-gray-200 rounded-lg p-4">
                                        <h4 className="font-bold text-gray-800 mb-4">💬 Interview Transcript</h4>
                                        
                                        <div className="space-y-4 max-h-96 overflow-y-auto">
                                          {interview.messages && interview.messages.length > 0 ? (
                                            interview.messages.map((msg, idx) => (
                                              <div key={idx} className={`flex ${msg.role === 'candidate' ? 'justify-end' : 'justify-start'}`}>
                                                <div className={`max-w-xs lg:max-w-md xl:max-w-lg px-4 py-2 rounded-lg ${
                                                  msg.role === 'candidate' 
                                                    ? 'bg-blue-500 text-white rounded-br-none' 
                                                    : 'bg-white border border-gray-300 text-gray-800 rounded-bl-none'
                                                }`}>
                                                  <p className="text-sm leading-relaxed">{msg.content}</p>
                                                </div>
                                              </div>
                                            ))
                                          ) : (
                                            <p className="text-gray-500 text-sm text-center py-4">No interview messages recorded</p>
                                          )}
                                        </div>
                                      </div>

                                      {/* Actions */}
                                      <div className="md:col-span-2 bg-red-50 border border-red-200 rounded-lg p-4">
                                        <h4 className="font-bold text-red-800 mb-3">⚙️ Actions</h4>
                                        <button
                                          onClick={() => handleDeleteInterview(interview.id, interview.candidate_name)}
                                          className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 font-medium"
                                        >
                                          🗑️ Delete Interview
                                        </button>
                                      </div>
                                    </div>
                                  </td>
                                </tr>
                              )}
                            </React.Fragment>
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
