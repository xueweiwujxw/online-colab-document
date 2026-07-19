import { getJSON, sendJSON } from './client';

export type DocumentPermission = {
  id: string;
  documentId: string;
  subjectType: string;
  subjectId: string;
  subjectDisplayName?: string;
  subjectEmail?: string;
  permission: 'viewer' | 'editor';
  createdAt: string;
};

export type GrantPermissionInput = {
  subjectType: 'user';
  subjectId: string;
  permission: 'viewer' | 'editor';
};

export async function listPermissions(documentId: string): Promise<DocumentPermission[]> {
  const response = await getJSON<{ items: DocumentPermission[] }>(
    `/api/documents/${documentId}/permissions`,
  );
  return response.items;
}

export function grantPermission(
  documentId: string,
  input: GrantPermissionInput,
): Promise<DocumentPermission> {
  return sendJSON<DocumentPermission>(`/api/documents/${documentId}/permissions`, input);
}

export function deletePermission(
  documentId: string,
  permissionId: string,
): Promise<{ status: string }> {
  return sendJSON<{ status: string }>(
    `/api/documents/${documentId}/permissions/${permissionId}`,
    undefined,
    { method: 'DELETE' },
  );
}
