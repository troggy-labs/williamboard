'use client'

import { useState, useCallback } from 'react'
import { useDropzone } from 'react-dropzone'

interface UploadStatus {
  submissionId?: string
  status: 'idle' | 'uploading' | 'processing' | 'completed' | 'error'
  message?: string
  eventsFound?: number
}

export function FileUpload() {
  const [uploadStatus, setUploadStatus] = useState<UploadStatus>({ status: 'idle' })

  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    if (acceptedFiles.length === 0) return

    const file = acceptedFiles[0]
    setUploadStatus({ status: 'uploading', message: 'Preparing upload...' })

    try {
      // Step 1: Get signed URL - call backend directly
      const signedUrlResponse = await fetch('http://localhost:8080/v1/uploads/signed-url', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ contentType: file.type }),
      })

      if (!signedUrlResponse.ok) {
        throw new Error('Failed to get upload URL')
      }

      const { uploadUrl, submissionId } = await signedUrlResponse.json()

      setUploadStatus({
        status: 'uploading',
        message: 'Processing with AI...',
        submissionId,
      })

      // Step 2: Upload file - call backend directly with extended timeout
      const formData = new FormData()
      formData.append('file', file)

      const uploadResponse = await fetch(`http://localhost:8080/v1/uploads/${submissionId}`, {
        method: 'PUT',
        body: formData,
        // Extended timeout for AI processing
        signal: AbortSignal.timeout(120000), // 2 minutes
      })

      if (!uploadResponse.ok) {
        throw new Error('Upload failed')
      }

      const result = await uploadResponse.json()
      
      setUploadStatus({
        status: 'completed',
        message: `Processing complete! Found ${result.eventsFound} events.`,
        submissionId,
        eventsFound: result.eventsFound,
      })

    } catch (error) {
      console.error('Upload error:', error)
      setUploadStatus({
        status: 'error',
        message: error instanceof Error ? error.message : 'Upload failed',
      })
    }
  }, [])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'image/*': ['.jpeg', '.jpg', '.png', '.webp'],
    },
    maxFiles: 1,
    disabled: uploadStatus.status === 'uploading' || uploadStatus.status === 'processing',
  })

  const resetUpload = () => {
    setUploadStatus({ status: 'idle' })
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div
        {...getRootProps()}
        className={`
          border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors
          ${isDragActive ? 'border-blue-400 bg-blue-50' : 'border-gray-300'}
          ${uploadStatus.status === 'uploading' || uploadStatus.status === 'processing' 
            ? 'opacity-50 cursor-not-allowed' : 'hover:border-blue-400 hover:bg-blue-50'}
        `}
      >
        <input {...getInputProps()} />
        
        {uploadStatus.status === 'idle' && (
          <>
            <svg
              className="mx-auto h-12 w-12 text-gray-400"
              stroke="currentColor"
              fill="none"
              viewBox="0 0 48 48"
            >
              <path
                d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02"
                strokeWidth={2}
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
            <p className="mt-2 text-sm text-gray-600">
              {isDragActive
                ? 'Drop the bulletin board image here'
                : 'Drop a bulletin board image here, or click to select'}
            </p>
            <p className="text-xs text-gray-500">PNG, JPG, WEBP up to 12MB</p>
          </>
        )}

        {uploadStatus.status === 'uploading' && (
          <>
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-2 text-sm text-gray-600">{uploadStatus.message}</p>
          </>
        )}

        {uploadStatus.status === 'processing' && (
          <>
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-2 text-sm text-gray-600">Processing with AI...</p>
          </>
        )}

        {uploadStatus.status === 'completed' && (
          <>
            <svg
              className="mx-auto h-12 w-12 text-green-500"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M5 13l4 4L19 7"
              />
            </svg>
            <p className="mt-2 text-sm text-green-600">{uploadStatus.message}</p>
            {uploadStatus.submissionId && (
              <div className="mt-4 p-4 bg-green-50 rounded-lg">
                <h3 className="font-medium text-green-800">Success!</h3>
                <p className="text-sm text-green-700 mt-1">
                  Submission ID: {uploadStatus.submissionId}
                </p>
                {uploadStatus.eventsFound && uploadStatus.eventsFound > 0 && (
                  <p className="text-sm text-green-700">
                    Found {uploadStatus.eventsFound} events in your image.
                  </p>
                )}
              </div>
            )}
            <button
              onClick={resetUpload}
              className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
            >
              Upload Another Image
            </button>
          </>
        )}

        {uploadStatus.status === 'error' && (
          <>
            <svg
              className="mx-auto h-12 w-12 text-red-500"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
            <p className="mt-2 text-sm text-red-600">{uploadStatus.message}</p>
            <button
              onClick={resetUpload}
              className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
            >
              Try Again
            </button>
          </>
        )}
      </div>
    </div>
  )
}