'use client'

import { useState, useEffect } from 'react'

interface Event {
  id: string
  title: string
  start_ts: string
  end_ts?: string
  venue_name?: string
  address?: string
  description?: string
  url?: string
  price?: string
  organizer?: string
  source: string
}

export function EventList() {
  const [allEvents, setAllEvents] = useState<Event[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchEvents = async () => {
      try {
        // Call backend directly and get all events (upcoming and past)
        const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080'
        const response = await fetch(`${apiBaseUrl}/v1/events?limit=500&include_past=true`)
        if (response.ok) {
          const data = await response.json()
          const eventData = data.features?.map((feature: any) => ({
            id: feature.id,
            title: feature.properties.title,
            start_ts: feature.properties.start_ts,
            end_ts: feature.properties.end_ts,
            venue_name: feature.properties.venue_name,
            address: feature.properties.address,
            description: feature.properties.description,
            url: feature.properties.url,
            price: feature.properties.price,
            organizer: feature.properties.organizer,
            source: feature.properties.source,
          })) || []
          setAllEvents(eventData)
        }
      } catch (error) {
        console.error('Failed to fetch events:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchEvents()
  }, [])

  // Group events into upcoming and past
  const now = new Date()
  const upcomingEvents = allEvents.filter(event => new Date(event.start_ts) >= now)
  const pastEvents = allEvents.filter(event => new Date(event.start_ts) < now)

  // Sort events by date
  upcomingEvents.sort((a, b) => new Date(a.start_ts).getTime() - new Date(b.start_ts).getTime())
  pastEvents.sort((a, b) => new Date(b.start_ts).getTime() - new Date(a.start_ts).getTime())

  const downloadICS = async (eventId: string) => {
    try {
      const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080'
      const response = await fetch(`${apiBaseUrl}/v1/events/${eventId}/ics`)
      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `event-${eventId}.ics`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        window.URL.revokeObjectURL(url)
      }
    } catch (error) {
      console.error('Failed to download calendar file:', error)
    }
  }

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleDateString('en-US', {
      weekday: 'short',
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
    })
  }

  const renderEventCard = (event: Event) => (
    <div
      key={event.id}
      className="bg-white rounded-lg shadow-sm border p-6 hover:shadow-md transition-shadow"
    >
      <div className="flex justify-between items-start mb-4">
        <h3 className="text-lg font-semibold text-gray-900 line-clamp-2">
          {event.title}
        </h3>
      </div>

      <div className="space-y-2 text-sm text-gray-600">
        <div className="flex items-center">
          <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          {formatDate(event.start_ts)}
        </div>
        
        {event.venue_name && (
          <div className="flex items-center">
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            {event.venue_name}
          </div>
        )}

        {event.address && (
          <div className="flex items-start">
            <svg className="w-4 h-4 mr-2 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
            <span className="line-clamp-2">{event.address}</span>
          </div>
        )}

        {event.price && (
          <div className="flex items-center">
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1" />
            </svg>
            {event.price}
          </div>
        )}

        {event.organizer && (
          <div className="flex items-center">
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
            </svg>
            {event.organizer}
          </div>
        )}
      </div>

      {event.description && (
        <p className="text-sm text-gray-600 mt-3 line-clamp-3">
          {event.description}
        </p>
      )}

      <div className="mt-4 pt-4 border-t border-gray-100 flex justify-between items-center">
        <button
          onClick={() => downloadICS(event.id)}
          className="text-blue-600 hover:text-blue-800 text-sm font-medium flex items-center"
        >
          <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          Add to Calendar
        </button>
        {event.url && (
          <a 
            href={event.url} 
            target="_blank" 
            rel="noopener noreferrer"
            className="text-blue-600 hover:text-blue-800 text-sm font-medium flex items-center"
          >
            <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            View Details
          </a>
        )}
      </div>
    </div>
  )

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (allEvents.length === 0) {
    return (
      <div className="text-center py-16">
        <svg
          className="mx-auto h-16 w-16 text-gray-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
          />
        </svg>
        <h3 className="text-lg font-medium text-gray-900 mt-4">No Events Found</h3>
        <p className="text-gray-600 mt-2">
          Upload some bulletin board images to discover events!
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-12">
      {/* Upcoming Events Section */}
      {upcomingEvents.length > 0 && (
        <div>
          <div className="flex items-center mb-6">
            <h2 className="text-2xl font-bold text-gray-900">Upcoming Events</h2>
            <span className="ml-3 bg-blue-100 text-blue-800 text-sm font-medium px-3 py-1 rounded-full">
              {upcomingEvents.length}
            </span>
          </div>
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {upcomingEvents.map(renderEventCard)}
          </div>
        </div>
      )}

      {/* Past Events Section */}
      {pastEvents.length > 0 && (
        <div>
          <div className="flex items-center mb-6">
            <h2 className="text-2xl font-bold text-gray-900">Past Events</h2>
            <span className="ml-3 bg-gray-100 text-gray-800 text-sm font-medium px-3 py-1 rounded-full">
              {pastEvents.length}
            </span>
          </div>
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {pastEvents.slice(0, 12).map(renderEventCard)}
          </div>
          {pastEvents.length > 12 && (
            <div className="text-center mt-6">
              <button className="text-gray-600 hover:text-gray-800 text-sm font-medium">
                Show {pastEvents.length - 12} more past events
              </button>
            </div>
          )}
        </div>
      )}

      {/* Show message if no upcoming events */}
      {upcomingEvents.length === 0 && pastEvents.length > 0 && (
        <div className="text-center py-8 bg-blue-50 rounded-lg">
          <svg
            className="mx-auto h-12 w-12 text-blue-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
            />
          </svg>
          <h3 className="text-lg font-medium text-gray-900 mt-4">No Upcoming Events</h3>
          <p className="text-gray-600 mt-2">
            Check back soon for new events, or upload bulletin board images to discover more!
          </p>
        </div>
      )}
    </div>
  )
}