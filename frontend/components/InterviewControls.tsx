'use client'

interface InterviewControlsProps {
  onFinish: () => void
  canFinish: boolean
  questionsAsked: number
}

export default function InterviewControls({
  onFinish,
  canFinish,
  questionsAsked,
}: InterviewControlsProps) {
  return (
    <div className="mt-8 p-4 bg-gray-50 rounded-lg flex justify-between items-center">
      <div className="text-gray-700">
        <p className="font-semibold">
          Questions Asked: {questionsAsked} / 6-8
        </p>
        <p className="text-sm text-gray-600">
          {questionsAsked < 3
            ? 'Answer at least 3 questions before finishing'
            : 'You can finish the interview now'}
        </p>
      </div>

      <button
        onClick={onFinish}
        disabled={!canFinish}
        className="px-6 py-2 bg-green-600 text-white font-semibold rounded-lg hover:bg-green-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
      >
        Finish Interview
      </button>
    </div>
  )
}
