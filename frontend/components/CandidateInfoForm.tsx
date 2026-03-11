'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import axios from 'axios'

export default function CandidateInfoForm() {
  const router = useRouter()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [successMessage, setSuccessMessage] = useState('')
  const [interviewId, setInterviewId] = useState<string | null>(null)
  const [documents, setDocuments] = useState<File[]>([])
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    role: '',
  })

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target
    setFormData(prev => ({ ...prev, [name]: value }))
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      setDocuments(Array.from(e.target.files))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccessMessage('')

    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const response = await axios.post(`${apiUrl}/interview/start`, formData)
      const newInterviewId = response.data.interview_id
      setInterviewId(newInterviewId)
      setSuccessMessage('Interview started! You can now upload documents or proceed to interview.')
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to start interview')
    } finally {
      setLoading(false)
    }
  }

  const handleUploadDocuments = async () => {
    if (!interviewId || documents.length === 0) {
      setError('Please select at least one document')
      return
    }

    setLoading(true)
    setError('')

    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      
      for (const doc of documents) {
        const formDataUpload = new FormData()
        formDataUpload.append('document', doc)
        
        await axios.post(
          `${apiUrl}/interview/${interviewId}/upload-document`,
          formDataUpload,
          {
            headers: {
              'Content-Type': 'multipart/form-data',
            },
          }
        )
      }

      setSuccessMessage(`${documents.length} document(s) uploaded successfully! You can upload more documents or start the interview.`)
      setDocuments([])
      
      // Reset file input to allow uploading more documents
      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement
      if (fileInput) {
        fileInput.value = ''
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to upload documents')
    } finally {
      setLoading(false)
    }
  }

  const handleProceedToInterview = () => {
    if (interviewId) {
      router.push(`/interview/${interviewId}`)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-lg p-8 w-full max-w-md">
        <h1 className="text-3xl font-bold text-gray-800 mb-2">VoxHire AI</h1>
        <p className="text-gray-600 mb-8">Start your interview</p>

        {error && (
          <div className="mb-4 p-4 bg-red-100 text-red-700 rounded-lg">
            {error}
          </div>
        )}

        {successMessage && (
          <div className="mb-4 p-4 bg-green-100 text-green-700 rounded-lg">
            {successMessage}
          </div>
        )}

        {!interviewId ? (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Full Name
              </label>
              <input
                type="text"
                name="name"
                value={formData.name}
                onChange={handleChange}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="John Doe"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Email
              </label>
              <input
                type="email"
                name="email"
                value={formData.email}
                onChange={handleChange}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="john@example.com"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Position
              </label>
              <select
                name="role"
                value={formData.role}
                onChange={handleChange}
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Select a position</option>
                <option value="software_engineer">Software Engineer</option>
                <option value="product_manager">Product Manager</option>
                <option value="data_scientist">Data Scientist</option>
                <option value="designer">Designer</option>
                <option value="other">Other</option>
              </select>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2 px-4 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-400 font-medium transition"
            >
              {loading ? 'Starting...' : 'Start Interview'}
            </button>
          </form>
        ) : (
          <div className="space-y-4">
            <p className="text-sm text-gray-600 font-medium mb-2">
              Interview Details
            </p>
            <div className="bg-gray-50 p-3 rounded text-sm">
              <p><strong>Candidate:</strong> {formData.name}</p>
              <p><strong>Email:</strong> {formData.email}</p>
              <p><strong>Position:</strong> {formData.role}</p>
            </div>

            <div className="border-t pt-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Upload Documents (Optional)
              </label>
              <p className="text-xs text-gray-600 mb-3">
                You can upload your resume, Aadhar, PAN, or other documents. You can upload multiple times.
              </p>
              <input
                type="file"
                multiple
                onChange={handleFileChange}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 cursor-pointer"
                accept=".pdf,.jpg,.jpeg,.png,.doc,.docx"
              />
            </div>

            {documents.length > 0 && (
              <div className="bg-blue-50 p-3 rounded border border-blue-200">
                <p className="text-sm font-medium text-gray-700 mb-2">
                  Ready to upload ({documents.length}):
                </p>
                <ul className="text-sm text-gray-600">
                  {documents.map((doc, idx) => (
                    <li key={idx}>• {doc.name}</li>
                  ))}
                </ul>
              </div>
            )}

            {documents.length > 0 && (
              <button
                onClick={handleUploadDocuments}
                disabled={loading}
                className="w-full py-2 px-4 bg-green-500 text-white rounded-lg hover:bg-green-600 disabled:bg-gray-400 font-medium transition"
              >
                {loading ? 'Uploading...' : `Upload ${documents.length} Document(s)`}
              </button>
            )}

            <button
              onClick={handleProceedToInterview}
              disabled={loading}
              className="w-full py-2 px-4 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:bg-gray-400 font-medium transition"
            >
              Start Interview Now
            </button>

            {successMessage && (
              <p className="text-xs text-green-600 text-center">
                {successMessage}
              </p>
            )}
          </div>
        )}

        <p className="text-xs text-gray-500 mt-6 text-center">
          This is a chat-based interview. Respond naturally to questions.
        </p>
      </div>
    </div>
  )
}
