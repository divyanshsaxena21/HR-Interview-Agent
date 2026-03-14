'use client'

import { useState, useEffect } from 'react'
import axios from 'axios'

interface Candidate {
  _id: string
  name: string
  email: string
  phone?: string
  role: string
  github?: string
  linkedin?: string
  status: string
  created_at: string
}

interface CandidateListProps {
  refreshTrigger?: number
}

export default function CandidateList({ refreshTrigger }: CandidateListProps) {
  const [candidates, setCandidates] = useState<Candidate[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchCandidates()
  }, [refreshTrigger])

  const fetchCandidates = async () => {
    try {
      setLoading(true)
      const token = localStorage.getItem('token')
      const response = await axios.get(
        `${process.env.NEXT_PUBLIC_API_URL}/admin/candidates`,
        {
          headers: { 'Authorization': `Bearer ${token}` },
        }
      )
      setCandidates(response.data || [])
      setError(null)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load candidates')
    } finally {
      setLoading(false)
    }
  }

  const screenCandidate = async (candidateId: string) => {
    try {
      const token = localStorage.getItem('token')
      const response = await axios.post(
        `${process.env.NEXT_PUBLIC_API_URL}/admin/candidates/${candidateId}/screen`,
        {},
        {
          headers: { 'Authorization': `Bearer ${token}` },
        }
      )
      alert(`Candidate screened and interview scheduled!\nSession: ${response.data.session_id}`)
      fetchCandidates()
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to screen candidate')
    }
  }

  const deleteCandidate = async (candidateId: string, candidateName: string) => {
    if (!window.confirm(`Are you sure you want to delete ${candidateName}?`)) {
      return
    }
    
    try {
      const token = localStorage.getItem('token')
      await axios.delete(
        `${process.env.NEXT_PUBLIC_API_URL}/admin/candidates/${candidateId}`,
        {
          headers: { 'Authorization': `Bearer ${token}` },
        }
      )
      alert('Candidate deleted successfully')
      fetchCandidates()
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to delete candidate')
    }
  }

  if (loading) return <div className="text-center py-4">Loading candidates...</div>
  if (error) return <div className="text-red-600 py-4">{error}</div>

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <div className="px-6 py-4 border-b border-gray-200">
        <h2 className="text-xl font-semibold">Candidates ({candidates.length})</h2>
      </div>

      {candidates.length === 0 ? (
        <div className="p-6 text-center text-gray-500">
          No candidates yet. Import some to get started.
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left font-semibold text-gray-700">Name</th>
                <th className="px-6 py-3 text-left font-semibold text-gray-700">Email</th>
                <th className="px-6 py-3 text-left font-semibold text-gray-700">Role</th>
                <th className="px-6 py-3 text-left font-semibold text-gray-700">Status</th>
                <th className="px-6 py-3 text-left font-semibold text-gray-700">Actions</th>
              </tr>
            </thead>
            <tbody>
              {candidates.map((candidate) => (
                <tr key={candidate._id} className="border-b border-gray-200 hover:bg-gray-50">
                  <td className="px-6 py-4 font-medium">{candidate.name}</td>
                  <td className="px-6 py-4">{candidate.email}</td>
                  <td className="px-6 py-4">{candidate.role}</td>
                  <td className="px-6 py-4">
                    <span className={`inline-block px-3 py-1 rounded-full text-xs font-semibold ${
                      candidate.status === 'pending_screening' ? 'bg-yellow-100 text-yellow-800' :
                      candidate.status === 'screened' ? 'bg-blue-100 text-blue-800' :
                      candidate.status === 'interviewed' ? 'bg-green-100 text-green-800' :
                      'bg-gray-100 text-gray-800'
                    }`}>
                      {candidate.status}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex gap-2">
                      {candidate.status === 'pending_screening' && (
                        <button
                          onClick={() => screenCandidate(candidate._id)}
                          className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 text-xs font-semibold"
                        >
                          Screen
                        </button>
                      )}
                      <button
                        onClick={() => deleteCandidate(candidate._id, candidate.name)}
                        className="bg-red-600 text-white px-4 py-2 rounded hover:bg-red-700 text-xs font-semibold"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
