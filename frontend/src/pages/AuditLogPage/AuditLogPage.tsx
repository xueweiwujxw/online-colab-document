import { FormEvent, useEffect, useState } from 'react';

import { listAuditLogs, type AuditLog, type AuditLogFilter } from '../../api/audit';
import { errorMessage } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';

type AuditState =
  | { status: 'loading'; items: AuditLog[]; error: null }
  | { status: 'success'; items: AuditLog[]; error: null }
  | { status: 'error'; items: AuditLog[]; error: string };

const defaultFilter: AuditLogFilter = { limit: 50 };

export function AuditLogPage() {
  const auth = useAuth();
  const [filter, setFilter] = useState<AuditLogFilter>(defaultFilter);
  const [state, setState] = useState<AuditState>({
    status: 'loading',
    items: [],
    error: null,
  });

  async function refresh(nextFilter = filter) {
    setState((current) => ({ status: 'loading', items: current.items, error: null }));
    try {
      const items = await listAuditLogs(nextFilter);
      setState({ status: 'success', items, error: null });
    } catch (error) {
      setState({
        status: 'error',
        items: [],
        error: errorMessage(error, 'Failed to load audit logs'),
      });
    }
  }

  useEffect(() => {
    if (auth.status === 'authenticated' && auth.user.isAdmin) {
      void refresh(defaultFilter);
    }
  }, [auth.status, auth.user?.isAdmin]);

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void refresh(filter);
  }

  if (auth.status === 'loading') {
    return (
      <main className="app-shell">
        <section className="empty-state">Loading</section>
      </main>
    );
  }

  if (auth.status === 'anonymous') {
    window.location.replace('/login');
    return null;
  }

  if (!auth.user.isAdmin) {
    return (
      <main className="app-shell">
        <a className="back-link" href="/documents">
          Back to documents
        </a>
        <section className="empty-state">Forbidden</section>
      </main>
    );
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Admin</p>
          <h1>Audit logs</h1>
        </div>
        <div className="user-actions">
          <a className="secondary-button" href="/documents">
            Documents
          </a>
          <button className="secondary-button" onClick={() => void auth.logout()} type="button">
            Sign out
          </button>
        </div>
      </header>
      <form className="audit-filters" onSubmit={onSubmit}>
        <label className="field">
          Action
          <input
            onChange={(event) => setFilter({ ...filter, action: event.target.value })}
            placeholder="document.upload"
            value={filter.action ?? ''}
          />
        </label>
        <label className="field">
          Target type
          <input
            onChange={(event) => setFilter({ ...filter, targetType: event.target.value })}
            placeholder="document"
            value={filter.targetType ?? ''}
          />
        </label>
        <label className="field">
          Actor user ID
          <input
            onChange={(event) => setFilter({ ...filter, actorUserId: event.target.value })}
            value={filter.actorUserId ?? ''}
          />
        </label>
        <label className="field">
          Limit
          <input
            min="1"
            onChange={(event) => setFilter({ ...filter, limit: Number(event.target.value) })}
            type="number"
            value={filter.limit ?? 50}
          />
        </label>
        <button className="secondary-button audit-filter-button" type="submit">
          Filter
        </button>
      </form>
      {state.status === 'loading' ? <section className="empty-state">Loading</section> : null}
      {state.status === 'error' ? <section className="empty-state">{state.error}</section> : null}
      {state.status === 'success' && state.items.length === 0 ? (
        <section className="empty-state">No audit logs yet.</section>
      ) : null}
      {state.status === 'success' && state.items.length > 0 ? (
        <section className="audit-list">
          {state.items.map((item) => (
            <article className="audit-row" key={item.id}>
              <span className="document-meta">{formatDate(item.createdAt)}</span>
              <span className="document-meta">{item.actorUserId ?? 'anonymous'}</span>
              <strong>{item.action}</strong>
              <span className="document-meta">
                {item.targetType}:{item.targetId}
              </span>
              <span className="document-meta">{item.ipAddr ?? '-'}</span>
              <span className="document-meta audit-user-agent">{item.userAgent ?? '-'}</span>
              <code>{JSON.stringify(item.metadata)}</code>
            </article>
          ))}
        </section>
      ) : null}
    </main>
  );
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}
