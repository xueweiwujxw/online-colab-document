import { ApiError, apiBaseUrl, getJSON, sendJSON } from './client';

export type DocumentItem = {
  id: string;
  title: string;
  originalFilename: string;
  fileExt: string;
  mimeType: string;
  sizeBytes: number;
  updatedAt: string;
  createdAt: string;
  canManage: boolean;
  canEdit: boolean;
};

export type DocumentVersion = {
  id: string;
  versionNo: number;
  sizeBytes: number;
  createdBy: string | null;
  createdAt: string;
};

export type MarkdownDocument = {
  document: DocumentItem;
  content: string;
  canEdit: boolean;
};

export type PresenceUser = {
  userId: string;
  displayName: string;
  canEdit: boolean;
};

export type MarkdownSnapshot = {
  documentId: string;
  content: string;
  versionNo: number;
  canEdit: boolean;
  users: PresenceUser[];
};

export async function listDocuments(): Promise<DocumentItem[]> {
  const response = await getJSON<{ items: DocumentItem[] }>('/api/documents');
  return response.items;
}

export function getDocument(id: string): Promise<DocumentItem> {
  return getJSON<DocumentItem>(`/api/documents/${id}`);
}

export async function listDocumentVersions(id: string): Promise<DocumentVersion[]> {
  const response = await getJSON<{ items: DocumentVersion[] }>(`/api/documents/${id}/versions`);
  return response.items;
}

export function deleteDocument(id: string): Promise<{ status: string }> {
  return sendJSON<{ status: string }>(`/api/documents/${id}`, undefined, {
    method: 'DELETE',
  });
}

export async function uploadDocument(file: File): Promise<DocumentItem> {
  const formData = new FormData();
  formData.append('file', file);
  const response = await fetch(`${apiBaseUrl}/api/documents/upload`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
    body: formData,
  });
  if (!response.ok) {
    throw new ApiError(response.status);
  }
  return response.json() as Promise<DocumentItem>;
}

export function documentDownloadURL(id: string): string {
  return `${apiBaseUrl}/api/documents/${id}/download`;
}

export function documentVersionDownloadURL(id: string, versionId: string): string {
  return `${apiBaseUrl}/api/documents/${id}/versions/${versionId}/download`;
}

export function restoreDocumentVersion(id: string, versionId: string): Promise<DocumentVersion> {
  return sendJSON<DocumentVersion>(`/api/documents/${id}/versions/${versionId}/restore`);
}

export function getMarkdownDocument(id: string): Promise<MarkdownDocument> {
  return getJSON<MarkdownDocument>(`/api/documents/${id}/markdown`);
}

export function saveMarkdownDocument(id: string, content: string): Promise<MarkdownDocument> {
  return sendJSON<MarkdownDocument>(
    `/api/documents/${id}/markdown`,
    { content },
    { method: 'PUT' },
  );
}

export function getMarkdownSnapshot(id: string): Promise<MarkdownSnapshot> {
  return getJSON<MarkdownSnapshot>(`/api/documents/${id}/markdown/snapshot`);
}

export function markdownWebSocketURL(id: string): string {
  const base =
    apiBaseUrl === ''
      ? window.location.origin
      : apiBaseUrl;
  return `${base.replace(/^http/, 'ws')}/api/documents/${id}/markdown/ws`;
}
