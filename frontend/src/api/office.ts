import { ApiError, apiBaseUrl, getJSON } from './client';

export type OfficeSession = {
  provider: 'casual' | string;
  documentId: string;
  fileExt: string;
  title: string;
  mode: 'view' | 'edit';
  downloadUrl: string;
  saveUrl: string;
};

export type OfficeCollabSession = {
  enabled: boolean;
  documentId: string;
  fileExt: string;
  room: string;
  role: 'view' | 'write';
  serverUrl?: string;
  reason?: string;
};

export function getOfficeSession(documentId: string): Promise<OfficeSession> {
  return getJSON<OfficeSession>(`/api/documents/${documentId}/office/session`);
}

export function getOfficeCollabSession(documentId: string): Promise<OfficeCollabSession> {
  return getJSON<OfficeCollabSession>(`/api/documents/${documentId}/office/collab/session`);
}

export async function fetchOfficeContent(url: string): Promise<ArrayBuffer> {
  const response = await fetch(resolveApiUrl(url), {
    credentials: 'include',
    headers: {
      Accept: 'application/octet-stream',
    },
  });
  if (!response.ok) {
    throw new ApiError(response.status);
  }
  return response.arrayBuffer();
}

export async function saveOfficeContent(url: string, buffer: ArrayBuffer): Promise<{ etag: string }> {
  const response = await fetch(resolveApiUrl(url), {
    method: 'PUT',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/octet-stream',
    },
    body: buffer,
  });
  if (!response.ok) {
    throw new ApiError(response.status);
  }
  return response.json() as Promise<{ etag: string }>;
}

function resolveApiUrl(url: string): string {
  if (/^https?:\/\/(localhost|127\.0\.0\.1):8080\/api\//i.test(url)) {
    const parsed = new URL(url);
    return `${parsed.pathname}${parsed.search}`;
  }
  if (/^https?:\/\//i.test(url)) {
    return url;
  }
  return `${apiBaseUrl}${url}`;
}
