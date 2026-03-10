import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'AI Recruiter Agent',
  description: 'Automated AI-powered recruitment screening',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
