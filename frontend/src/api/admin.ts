import { getJSON, sendJSON } from './client';

export type AdminUser = { id: string; email: string; displayName: string; authSource: string; isAdmin: boolean; disabled: boolean; createdAt: string; avatarUrl?: string };
export type StorageObject = { key: string; sizeBytes: number; contentType: string; updatedAt: string };
export type StorageSummary = { objectCount: number; totalBytes: number };

export async function listAdminUsers(query = ''): Promise<AdminUser[]> { const q = new URLSearchParams({ limit: '50' }); if (query) q.set('q', query); return (await getJSON<{ items: AdminUser[] }>(`/api/admin/users?${q}`)).items; }
export function resetUserPassword(id: string, newPassword: string): Promise<{ status: string }> { return sendJSON(`/api/admin/users/${id}/password`, { newPassword }, { method: 'PUT' }); }
export async function getStorage(prefix = ''): Promise<{ usage: StorageSummary; items: StorageObject[] }> { const q = new URLSearchParams({ limit: '100' }); if (prefix) q.set('prefix', prefix); return getJSON(`/api/admin/storage?${q}`); }
export function deleteStorageObject(key: string): Promise<{ status: string }> { return sendJSON(`/api/admin/storage/object?${new URLSearchParams({ key })}`, undefined, { method: 'DELETE' }); }
