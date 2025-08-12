'use client'

import { useState, useEffect, useRef } from 'react'
import mapboxgl from 'mapbox-gl'
import 'mapbox-gl/dist/mapbox-gl.css'

// Set Mapbox access token - in production, this should be in environment variables
mapboxgl.accessToken = 'pk.eyJ1IjoidGVzdC11c2VyIiwiYSI6ImNsemJhYmNkZWZnaGkybXBmcGxxdGV0ZGkifQ.test' // Placeholder - replace with real token

interface Event {
  id: string
  title: string
  date?: string
  venue?: string
  address?: string
  latitude?: number
  longitude?: number
}

export function EventMap() {
  const [events, setEvents] = useState<Event[]>([])
  const [loading, setLoading] = useState(true)
  const [mapError, setMapError] = useState(false)
  const mapContainer = useRef<HTMLDivElement>(null)
  const map = useRef<mapboxgl.Map | null>(null)

  useEffect(() => {
    const fetchEvents = async () => {
      try {
        const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080'
        const response = await fetch(`${apiBaseUrl}/v1/events`)
        if (response.ok) {
          const data = await response.json()
          const eventData = data.features?.map((feature: any) => ({
            id: feature.id,
            title: feature.properties.title,
            date: feature.properties.date,
            venue: feature.properties.venue,
            address: feature.properties.address,
            latitude: feature.geometry?.coordinates?.[1],
            longitude: feature.geometry?.coordinates?.[0],
          })) || []
          setEvents(eventData)
        }
      } catch (error) {
        console.error('Failed to fetch events:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchEvents()
  }, [])

  useEffect(() => {
    if (!mapContainer.current || loading || mapError) return

    // Initialize map
    try {
      map.current = new mapboxgl.Map({
        container: mapContainer.current,
        style: 'mapbox://styles/mapbox/light-v11',
        center: [-122.4194, 37.7749], // Default to San Francisco
        zoom: 12
      })

      map.current.on('error', (e) => {
        console.error('Mapbox error:', e)
        setMapError(true)
      })

      // Add markers for events with location data
      const eventsWithLocation = events.filter(event => 
        event.latitude && event.longitude && 
        !isNaN(event.latitude) && !isNaN(event.longitude)
      )

      eventsWithLocation.forEach(event => {
        if (!map.current) return

        // Create popup content
        const popupContent = `
          <div class="p-3">
            <h3 class="font-semibold text-gray-900">${event.title}</h3>
            ${event.date ? `<p class="text-sm text-gray-600 mt-1">${event.date}</p>` : ''}
            ${event.venue ? `<p class="text-sm text-gray-600">${event.venue}</p>` : ''}
            ${event.address ? `<p class="text-sm text-gray-500">${event.address}</p>` : ''}
          </div>
        `

        const popup = new mapboxgl.Popup({ offset: 25 })
          .setHTML(popupContent)

        new mapboxgl.Marker({ color: '#2563eb' })
          .setLngLat([event.longitude!, event.latitude!])
          .setPopup(popup)
          .addTo(map.current)
      })

      // Fit map to show all markers if we have events
      if (eventsWithLocation.length > 0) {
        const bounds = new mapboxgl.LngLatBounds()
        eventsWithLocation.forEach(event => {
          bounds.extend([event.longitude!, event.latitude!])
        })
        map.current.fitBounds(bounds, { padding: 50 })
      }

    } catch (error) {
      console.error('Failed to initialize map:', error)
      setMapError(true)
    }

    // Cleanup
    return () => {
      if (map.current) {
        map.current.remove()
        map.current = null
      }
    }
  }, [events, loading, mapError])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96 bg-gray-100 rounded-lg">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (mapError) {
    return (
      <div className="bg-white rounded-lg shadow p-6">
        <div className="h-96 bg-gray-100 rounded-lg flex items-center justify-center">
          <div className="text-center">
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
                d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"
              />
            </svg>
            <h3 className="text-lg font-medium text-gray-900 mt-4">Map Unavailable</h3>
            <p className="text-gray-600 mt-2">
              Unable to load interactive map
              <br />
              <span className="text-sm">(Mapbox API key required)</span>
            </p>
          </div>
        </div>
        
        {events.length > 0 && (
          <div className="mt-6">
            <h4 className="text-sm font-medium text-gray-900 mb-3">Recent Events ({events.length})</h4>
            <div className="space-y-3">
              {events.slice(0, 5).map((event) => (
                <div key={event.id} className="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
                  <div className="flex-shrink-0">
                    <div className="w-2 h-2 bg-blue-500 rounded-full mt-2"></div>
                  </div>
                  <div className="flex-1">
                    <p className="font-medium text-gray-900">{event.title}</p>
                    {event.date && (
                      <p className="text-sm text-gray-600">{event.date}</p>
                    )}
                    {event.venue && (
                      <p className="text-sm text-gray-600">{event.venue}</p>
                    )}
                    {event.address && (
                      <p className="text-sm text-gray-500">{event.address}</p>
                    )}
                    {event.latitude && event.longitude && (
                      <p className="text-xs text-gray-400">
                        {event.latitude.toFixed(4)}, {event.longitude.toFixed(4)}
                      </p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="h-96 rounded-lg overflow-hidden">
        <div ref={mapContainer} className="w-full h-full" />
      </div>
      
      {events.length > 0 && (
        <div className="mt-6">
          <h4 className="text-sm font-medium text-gray-900 mb-3">
            Events on Map ({events.filter(e => e.latitude && e.longitude).length} of {events.length})
          </h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {events.filter(e => e.latitude && e.longitude).slice(0, 4).map((event) => (
              <div key={event.id} className="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
                <div className="flex-shrink-0">
                  <div className="w-3 h-3 bg-blue-500 rounded-full mt-1"></div>
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-gray-900 truncate">{event.title}</p>
                  {event.date && (
                    <p className="text-sm text-gray-600">{event.date}</p>
                  )}
                  {event.venue && (
                    <p className="text-sm text-gray-600 truncate">{event.venue}</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}