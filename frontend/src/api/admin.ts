import { apiBaseUrl, getJSON, sendJSON } from './client';

export type AdminUser = { id: string; email: string; displayName: string; authSource: string; isAdmin: boolean; disabled: boolean; createdAt: string; avatarUrl?: string };
export type AdminUserUpdate = { displayName?: string; email?: string; isAdmin?: boolean; disabled?: boolean };
export type StorageObject = { key: string; sizeBytes: number; contentType: string; updatedAt: string };
export type StorageSummary = { objectCount: number; totalBytes: number };
export type AdminDocument = { id: string; ownerId: string; title: string; originalFilename: string; fileExt: string; sizeBytes: number; updatedAt: string };
export type OIDCStatus = { enabled: boolean; issuerUrl: string; clientId: string; redirectUrl: string; scopes: string[]; autoMergeByEmail: boolean; secretConfigured: boolean; source: string };

export async function listAdminUsers(query = ''): Promise<AdminUser[]> { const q = new URLSearchParams({ limit: '100' }); if (query) q.set('q', query); return (await getJSON<{ items: AdminUser[] }>(`/api/admin/users?${q}`)).items; }
export function updateAdminUser(id: string, update: AdminUserUpdate): Promise<AdminUser> { return sendJSON(`/api/admin/users/${id}`, update, { method: 'PATCH' }); }
export function resetUserPassword(id: string, newPassword: string): Promise<{ status: string }> { return sendJSON(`/api/admin/users/${id}/password`, { newPassword }, { method: 'PUT' }); }
export async function getStorage(prefix = ''): Promise<{ usage: StorageSummary; items: StorageObject[] }> { const q = new URLSearchParams({ limit: '100' }); if (prefix) q.set('prefix', prefix); return getJSON(`/api/admin/storage?${q}`); }
export function deleteStorageObject(key: string): Promise<{ status: string }> { return sendJSON(`/api/admin/storage/object?${new URLSearchParams({ key })}`, undefined, { method: 'DELETE' }); }
export function storageObjectDownloadURL(key: string): string { return `${apiBaseUrl}/api/admin/storage/object?${new URLSearchParams({ key })}`; }
export async function listAdminDocuments(): Promise<AdminDocument[]> { return (await getJSON<{ items: AdminDocument[] }>('/api/admin/documents?limit=100')).items; }
export function deleteAdminDocument(id: string): Promise<{ status: string }> { return sendJSON(`/api/admin/documents/${id}`, undefined, { method: 'DELETE' }); }
export function getOIDCStatus(): Promise<OIDCStatus> { return getJSON('/api/admin/oidc'); }
