/**
 * OGA-app upload/download helpers.
 * Download URLs are resolved via the OGA service's proxy endpoint because
 * OGA users authenticate against a different identity provider whose tokens
 * are not valid for the main backend's upload API.
 */
import type { ApiClient } from '../api'

export interface UploadResponse {
  key: string
  name: string
}

export async function uploadFile(_apiClient: ApiClient, _file: File): Promise<UploadResponse> {
  throw new Error('File upload is not supported in the OGA portal')
}

export async function getDownloadUrl(apiClient: ApiClient, key: string): Promise<{ url: string; expiresAt: number }> {
  const response = await apiClient.get<{ download_url: string; expires_at: number }>(
    `/api/oga/uploads/${key}`
  )
  return { url: response.download_url, expiresAt: response.expires_at }
}
