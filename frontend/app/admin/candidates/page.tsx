'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import FileUploader from '@/components/FileUploader'
import CandidateList from '@/components/CandidateList'

export default function CandidatesPage() {
  const router = useRouter()
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [successMessage, setSuccessMessage] = useState('')
  const [refreshTrigger, setRefreshTrigger] = useState(0)

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (!token) {
      router.push('/admin/login')
      return
    }
    setIsAuthenticated(true)
  }, [router])

  const handleImportSuccess = (count: number) => {
    setSuccessMessage(`Successfully imported ${count} candidates`)
    setRefreshTrigger(prev => prev + 1)
    setTimeout(() => setSuccessMessage(''), 3000)
  }

  const handleImportError = (error: string) => {
    alert(`Error: ${error}`)
  }

  if (!isAuthenticated) {
    return <div className="text-center py-4">Redirecting to login...</div>
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100">
      <header className="bg-white border-b border-gray-200 shadow-sm">
        <div className="max-w-7xl mx-auto px-6 py-4 flex justify-between items-center">
          <h1 className="text-3xl font-bold text-gray-800">VoxHire AI - Admin Dashboard</h1>
          <div className="flex gap-3">
            <button
              onClick={() => router.push('/admin/dashboard')}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
            >
              ← Back to Dashboard
            </button>
            <button
              onClick={() => {
                localStorage.removeItem('token')
                router.push('/admin/login')
              }}
              className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
            >
              Logout
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        {successMessage && (
          <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg text-green-700">
            ✓ {successMessage}
          </div>
        )}

        <div className="grid gap-8">
          <FileUploader 
            onSuccess={handleImportSuccess}
            onError={handleImportError}
          />
          
          <CandidateList refreshTrigger={refreshTrigger} />
        </div>
      </main>
    </div>
  )
}
