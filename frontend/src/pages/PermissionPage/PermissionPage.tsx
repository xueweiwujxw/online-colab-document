import { FormEvent, useEffect, useState } from 'react';

import {
  deletePermission,
  grantPermission,
  listPermissions,
  type DocumentPermission,
} from '../../api/permissions';
import { useAuth } from '../../auth/AuthContext';

type PermissionState =
  | { status: 'loading'; items: DocumentPermission[]; error: null }
  | { status: 'success'; items: DocumentPermission[]; error: null }
  | { status: 'error'; items: DocumentPermission[]; error: string };

export function PermissionPage({ documentId }: { documentId: string }) {
  const auth = useAuth();
  const [state, setState] = useState<PermissionState>({
    status: 'loading',
    items: [],
    error: null,
  });
  const [subjectId, setSubjectId] = useState('');
  const [permission, setPermission] = useState<'viewer' | 'editor'>('viewer');
  const [actionError, setActionError] = useState<string | null>(null);

  async function refreshPermissions() {
    setState((current) => ({ status: 'loading', items: current.items, error: null }));
    try {
      const items = await listPermissions(documentId);
      setState({ status: 'success', items, error: null });
    } catch (error) {
      setState({
        status: 'error',
        items: [],
        error: error instanceof Error ? error.message : 'Failed to load permissions',
      });
    }
  }

  useEffect(() => {
    if (auth.status === 'authenticated') {
      void refreshPermissions();
    }
  }, [auth.status, documentId]);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setActionError(null);
    try {
      await grantPermission(documentId, {
        subjectType: 'user',
        subjectId,
        permission,
      });
      setSubjectId('');
      await refreshPermissions();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : 'Grant failed');
    }
  }

  async function onDelete(permissionId: string) {
    setActionError(null);
    try {
      await deletePermission(documentId, permissionId);
      await refreshPermissions();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : 'Delete failed');
    }
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

  return (
    <main className="app-shell">
      <a className="back-link" href={`/documents/${documentId}`}>
        Back to document
      </a>
      <header className="detail-header">
        <div>
          <p className="eyebrow">Permissions</p>
          <h1>Manage access</h1>
        </div>
      </header>

      <form className="permission-form" onSubmit={onSubmit}>
        <label className="field">
          <span>User ID</span>
          <input
            onChange={(event) => setSubjectId(event.target.value)}
            required
            type="text"
            value={subjectId}
          />
        </label>
        <label className="field">
          <span>Permission</span>
          <select
            onChange={(event) => setPermission(event.target.value as 'viewer' | 'editor')}
            value={permission}
          >
            <option value="viewer">viewer</option>
            <option value="editor">editor</option>
          </select>
        </label>
        <button className="primary-button permission-submit" type="submit">
          Grant
        </button>
      </form>

      {actionError ? <p className="form-error">{actionError}</p> : null}
      {state.status === 'loading' ? <section className="empty-state">Loading</section> : null}
      {state.status === 'error' ? <section className="empty-state">{state.error}</section> : null}
      {state.status === 'success' && state.items.length === 0 ? (
        <section className="empty-state">No permissions yet.</section>
      ) : null}
      {state.status === 'success' && state.items.length > 0 ? (
        <section className="document-list">
          {state.items.map((item) => (
            <article className="document-row permission-row" key={item.id}>
              <span className="document-title">{item.subjectId}</span>
              <span className="document-meta">{item.subjectType}</span>
              <span className="document-meta">{item.permission}</span>
              <button
                className="secondary-button danger-button"
                onClick={() => void onDelete(item.id)}
                type="button"
              >
                Delete
              </button>
            </article>
          ))}
        </section>
      ) : null}
    </main>
  );
}
