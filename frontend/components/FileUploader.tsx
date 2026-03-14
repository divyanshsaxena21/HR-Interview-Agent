'use client'

import { useState, useRef } from 'react'
import axios from 'axios'

interface FileUploaderProps {
  onSuccess: (count: number) => void
  onError: (error: string) => void
}

export default function FileUploader({ onSuccess, onError }: FileUploaderProps) {
  const [loading, setLoading] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0]
    if (selectedFile) {
      const ext = selectedFile.name.split('.').pop()?.toLowerCase()
      if (ext === 'csv' || ext === 'xlsx') {
        setFile(selectedFile)
      } else {
        onError('Please select a CSV or Excel file')
      }
    }
  }

  const handleUpload = async () => {
    if (!file) {
      onError('Please select a file')
      return
    }

    setLoading(true)
    try {
      const formData = new FormData()
      formData.append('file', file)

      const token = localStorage.getItem('token')
      const response = await axios.post(
        `${process.env.NEXT_PUBLIC_API_URL}/admin/candidates/import`,
        formData,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'multipart/form-data',
          },
        }
      )

      setFile(null)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
      onSuccess(response.data.count)
    } catch (error: any) {
      onError(error.response?.data?.error || 'Upload failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="bg-white p-6 rounded-lg border border-gray-200">
      <h2 className="text-xl font-semibold mb-4">Import Candidates</h2>
      
      <div className="space-y-4">
        <div className="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center">
          <input
            ref={fileInputRef}
            type="file"
            accept=".csv,.xlsx"
            onChange={handleFileChange}
            className="hidden"
            id="file-input"
          />
          <label htmlFor="file-input" className="cursor-pointer">
            <p className="text-gray-600 mb-2">
              {file ? file.name : 'Select a CSV or Excel file'}
            </p>
            <p className="text-sm text-gray-400">
              Drag and drop or click to select
            </p>
          </label>
        </div>

        <button
          onClick={handleUpload}
          disabled={!file || loading}
          className="w-full bg-blue-600 text-white font-semibold py-2 rounded-lg hover:bg-blue-700 disabled:bg-gray-400 transition-colors"
        >
          {loading ? 'Uploading...' : 'Upload'}
        </button>

        <div className="bg-gray-50 p-4 rounded text-sm text-gray-600">
          <p className="font-semibold mb-2">Expected columns:</p>
          <ul className="list-disc list-inside space-y-1">
            <li>name (required)</li>
            <li>email (required)</li>
            <li>phone</li>
            <li>role</li>
            <li>github</li>
            <li>linkedin</li>
            <li>resume</li>
          </ul>
        </div>
      </div>
    </div>
  )
}
