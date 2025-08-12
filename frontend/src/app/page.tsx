'use client'

import { useState } from 'react'
import { FileUpload } from '../components/FileUpload'
import { EventMap } from '../components/EventMap'
import { EventList } from '../components/EventList'

export default function Home() {
  const [activeTab, setActiveTab] = useState<'upload' | 'map' | 'events'>('upload')

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-4">
            <div className="flex items-center">
              <h1 className="text-2xl font-bold text-gray-900">WilliamBoard</h1>
              <span className="ml-2 text-sm text-gray-600">Event Discovery</span>
            </div>
            <nav className="flex space-x-4">
              <button
                onClick={() => setActiveTab('upload')}
                className={`px-4 py-2 rounded-md text-sm font-medium ${
                  activeTab === 'upload'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                Upload
              </button>
              <button
                onClick={() => setActiveTab('map')}
                className={`px-4 py-2 rounded-md text-sm font-medium ${
                  activeTab === 'map'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                Map
              </button>
              <button
                onClick={() => setActiveTab('events')}
                className={`px-4 py-2 rounded-md text-sm font-medium ${
                  activeTab === 'events'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                Events
              </button>
            </nav>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === 'upload' && (
          <div className="space-y-8">
            <div className="text-center">
              <h2 className="text-3xl font-bold text-gray-900">
                Discover Events from Bulletin Boards
              </h2>
              <p className="mt-4 text-lg text-gray-600">
                Upload a photo of a bulletin board and we&apos;ll extract event information using AI
              </p>
            </div>
            <FileUpload />
          </div>
        )}
        
        {activeTab === 'map' && (
          <div className="space-y-8">
            <div className="text-center">
              <h2 className="text-3xl font-bold text-gray-900">Event Map</h2>
              <p className="mt-4 text-lg text-gray-600">
                Explore events on the map
              </p>
            </div>
            <EventMap />
          </div>
        )}
        
        {activeTab === 'events' && (
          <div className="space-y-8">
            <div className="text-center">
              <h2 className="text-3xl font-bold text-gray-900">All Events</h2>
              <p className="mt-4 text-lg text-gray-600">
                Browse all discovered events
              </p>
            </div>
            <EventList />
          </div>
        )}
      </main>
    </div>
  )
}