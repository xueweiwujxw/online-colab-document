import { getJSON } from './client';

export type AuditLog = {
  id: string;
  actorUserId: string | null;
  actorDisplayName: string | null;
  actorEmail: string | null;
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
  ipAddr?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
};

export type AuditLogPage = { items: AuditLog[]; hasMore: boolean };

export async function listAuditLogs(filter: AuditLogFilter = {}): Promise<AuditLogPage> {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(filter)) {
    if (value !== undefined && value !== '') {
      params.set(key, String(value));
    }
  }
  const query = params.toString();
  return getJSON<AuditLogPage>(
    `/api/admin/audit-logs${query ? `?${query}` : ''}`,
  );
}
