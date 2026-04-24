/**
 * Trader-app–specific upload implementation. Points to this app's backend;
 * when the API or auth changes, only this file is updated.
 */
import { getEnv } from '../runtimeConfig'
import type { ApiClient } from './api'

const API_BASE_URL = (() => {
  const value = getEnv('VITE_API_BASE_URL')
  if (!value) {
    throw new Error('Missing required environment variable: VITE_API_BASE_URL')
  }
  return value
})()

interface UploadMetadataRequest {
  filename: string
  mime_type: string
  size: number
}

interface UploadMetadataResponse {
  key: string
  name: string
  upload_url: string
}

interface DownloadMetadataResponse {
  download_url: string
  expires_at: number
}

export interface UploadResponse {
  key: string
  name: string
}

export async function uploadFile(apiClient: ApiClient, file: File): Promise<UploadResponse> {
  const metadata = await apiClient.post<UploadMetadataRequest, UploadMetadataResponse>(
    '/uploads',
    {
      filename: file.name,
      mime_type: file.type || 'application/octet-stream',
      size: file.size,
    }
  )

  // Upload file bytes directly to the storage destination (presigned URL)
  const uploadResponse = await fetch(metadata.upload_url, {
    method: 'PUT',
    headers: {
      'Content-Type': file.type || 'application/octet-stream',
    },
    body: file,
  })

  if (!uploadResponse.ok) {
    const errorText = await uploadResponse.text()
    console.error(`Direct storage upload error ${uploadResponse.status}: ${errorText}`)
    throw new Error(`Failed to upload file to storage: ${uploadResponse.status} ${uploadResponse.statusText}`)
  }

  return { key: metadata.key, name: metadata.name }
}

export async function getDownloadUrl(apiClient: ApiClient, key: string): Promise<{ url: string; expiresAt: number }> {
  const response = await apiClient.get<DownloadMetadataResponse>(`/uploads/${key}`)

  // Normalize the URL if it's a relative path (common in local dev)
  const url = response.download_url.startsWith('/')
    ? new URL(API_BASE_URL).origin + response.download_url
    : response.download_url

  return { url, expiresAt: response.expires_at }
}
