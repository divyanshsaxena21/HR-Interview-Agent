'use client'

interface TranscriptViewerProps {
  messages: Array<{
    role: string
    content: string
    timestamp: number
  }>
}

export default function TranscriptViewer({
  messages,
}: TranscriptViewerProps) {
  return (
    <div>
      <h3 className="text-lg font-semibold text-gray-800 mb-4">Transcript</h3>
      <div className="space-y-3 max-h-96 overflow-y-auto">
        {messages.length === 0 ? (
          <p className="text-gray-500 italic">No messages yet</p>
        ) : (
          messages.map((msg, i) => (
            <div
              key={i}
              className={`p-3 rounded-lg ${
                msg.role === 'candidate'
                  ? 'bg-blue-100 text-blue-900'
                  : msg.role === 'ai' || msg.role === 'assistant'
                  ? 'bg-indigo-100 text-indigo-900'
                  : 'bg-gray-200 text-gray-900'
              }`}
            >
              <p className="font-semibold text-sm mb-1">
                {msg.role === 'candidate' ? 'You' : 'AI Recruiter'}
              </p>
              <p className="text-sm">{msg.content}</p>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
