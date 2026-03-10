'use client'

import { useState, useRef, useEffect } from 'react'

interface AudioPlayerProps {
  src: string
  autoPlay?: boolean
  onEnded?: () => void
}

export default function AudioPlayer({ src, autoPlay = false, onEnded }: AudioPlayerProps) {
  const [isPlaying, setIsPlaying] = useState(false)
  const audioRef = useRef<HTMLAudioElement | null>(null)

  useEffect(() => {
    if (audioRef.current && autoPlay) {
      audioRef.current.play().catch(() => {})
    }
  }, [src, autoPlay])

  return (
    <div className="flex items-center gap-4 p-4 bg-blue-50 rounded-lg">
      <audio
        ref={audioRef}
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
        onEnded={() => onEnded && onEnded()}
        autoPlay={autoPlay}
        controls
        src={src}
        className="flex-1"
      />
    </div>
  )
}
