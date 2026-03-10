'use client'

interface FitBadgeProps {
  fit: string
}

export default function FitBadge({ fit }: FitBadgeProps) {
  let bgColor = 'bg-gray-100'
  let textColor = 'text-gray-700'

  if (fit === 'GOOD_FIT') {
    bgColor = 'bg-green-100'
    textColor = 'text-green-700'
  } else if (fit === 'POSSIBLE_FIT') {
    bgColor = 'bg-yellow-100'
    textColor = 'text-yellow-700'
  } else if (fit === 'NOT_FIT') {
    bgColor = 'bg-red-100'
    textColor = 'text-red-700'
  }

  return (
    <span
      className={`px-3 py-1 rounded-full text-sm font-semibold ${bgColor} ${textColor}`}
    >
      {fit.replace(/_/g, ' ')}
    </span>
  )
}
