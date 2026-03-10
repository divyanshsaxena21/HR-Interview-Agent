'use client'

import CandidateInfoForm from '@/components/CandidateInfoForm'

export default function Home() {
  return (
    <main className="min-h-screen">
      <CandidateInfoForm />
      <div className="fixed bottom-6 right-6">
        <a href="/admin/login" className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition text-sm font-medium">
          Admin Login
        </a>
      </div>
    </main>
  )
}
