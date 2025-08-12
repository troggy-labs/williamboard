import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'WilliamBoard - Event Discovery',
  description: 'Upload bulletin board photos to discover events',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className="antialiased">
        {children}
      </body>
    </html>
  )
}