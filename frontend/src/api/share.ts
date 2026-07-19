import { apiBaseUrl, getJSON, sendJSON } from './client';
import type { DocumentItem } from './documents';

export type SharePermission = 'viewer' | 'editor';

export type ShareLink = {
  id: string;
  documentId: string;
  permission: SharePermission;
  expiresAt: string | null;
  disabled: boolean;
  createdAt: string;
};

export type CreatedShareLink = ShareLink & {
  token: string;
  url: string;
};

export type ShareAccess = {
  document: DocumentItem;
  content?: string;
  canEdit: boolean;
};

export async function listShareLinks(documentId: string): Promise<ShareLink[]> {
  const response = await getJSON<{ items: ShareLink[] }>(`/api/documents/${documentId}/share-links`);
  return response.items;
}

export function createShareLink(
  documentId: string,
  input: { permission: SharePermission; expiresAt?: string | null },
): Promise<CreatedShareLink> {
  return sendJSON<CreatedShareLink>(`/api/documents/${documentId}/share-links`, input);
}

export function disableShareLink(id: string): Promise<{ status: string }> {
  return sendJSON<{ status: string }>(`/api/share-links/${id}`, undefined, {
    method: 'DELETE',
  });
}

export function getShareAccess(token: string): Promise<ShareAccess> {
  return getJSON<ShareAccess>(`/api/share/${encodeURIComponent(token)}`);
}

export function saveSharedMarkdown(token: string, content: string): Promise<ShareAccess> {
  return sendJSON<ShareAccess>(
    `/api/share/${encodeURIComponent(token)}/markdown`,
    { content },
    { method: 'PUT' },
  );
}

export function sharedDownloadURL(token: string): string {
  return `${apiBaseUrl}/api/share/${encodeURIComponent(token)}/download`;
}
