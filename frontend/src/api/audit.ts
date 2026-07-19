import { getJSON } from './client';

export type AuditLog = {
  id: string;
  actorUserId: string | null;
  action: string;
  targetType: string;
  targetId: string;
  ipAddr: string | null;
  userAgent: string | null;
  metadata: Record<string, unknown>;
  createdAt: string;
};

export type AuditLogFilter = {
  actorUserId?: string;
  action?: string;
  targetType?: string;
  targetId?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
};

export async function listAuditLogs(filter: AuditLogFilter = {}): Promise<AuditLog[]> {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(filter)) {
    if (value !== undefined && value !== '') {
      params.set(key, String(value));
    }
  }
  const query = params.toString();
  const response = await getJSON<{ items: AuditLog[] }>(
    `/api/admin/audit-logs${query ? `?${query}` : ''}`,
  );
  return response.items;
}
