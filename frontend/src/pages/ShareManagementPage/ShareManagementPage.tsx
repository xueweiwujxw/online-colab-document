import { FormEvent, useEffect, useState } from 'react';

import {
  createShareLink,
  disableShareLink,
  listShareLinks,
  type CreatedShareLink,
  type ShareLink,
  type SharePermission,
} from '../../api/share';
import { errorMessage } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';

type ShareState =
  | { status: 'loading'; items: ShareLink[]; error: null }
  | { status: 'success'; items: ShareLink[]; error: null }
  | { status: 'error'; items: ShareLink[]; error: string };

export function ShareManagementPage({ documentId }: { documentId: string }) {
  const auth = useAuth();
  const [state, setState] = useState<ShareState>({ status: 'loading', items: [], error: null });
  const [permission, setPermission] = useState<SharePermission>('viewer');
  const [expiresAt, setExpiresAt] = useState('');
  const [created, setCreated] = useState<CreatedShareLink | null>(null);
  const [copied, setCopied] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  async function refreshLinks() {
    setState((current) => ({ status: 'loading', items: current.items, error: null }));
    try {
      const items = await listShareLinks(documentId);
      setState({ status: 'success', items, error: null });
    } catch (error) {
      setState({
        status: 'error',
        items: [],
        error: errorMessage(error, 'Failed to load share links'),
      });
    }
  }

  useEffect(() => {
    if (auth.status === 'authenticated') {
      void refreshLinks();
    }
  }, [auth.status, documentId]);

  async function onCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setActionError(null);
    setCreated(null);
    setCopied(false);
    try {
      const response = await createShareLink(documentId, {
        permission,
        expiresAt: expiresAt === '' ? null : new Date(expiresAt).toISOString(),
      });
      setCreated(response);
      setExpiresAt('');
      await refreshLinks();
    } catch (error) {
      setActionError(errorMessage(error, 'Create share link failed'));
    }
  }

  async function onDisable(id: string) {
    if (!window.confirm('Disable this share link?')) {
      return;
    }
    setActionError(null);
    try {
      await disableShareLink(id);
      await refreshLinks();
    } catch (error) {
      setActionError(errorMessage(error, 'Disable share link failed'));
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
          <p className="eyebrow">Share</p>
          <h1>Share links</h1>
        </div>
      </header>

      <form className="permission-form" onSubmit={onCreate}>
        <label className="field">
          <span>Permission</span>
          <select
            onChange={(event) => setPermission(event.target.value as SharePermission)}
            value={permission}
          >
            <option value="viewer">viewer</option>
            <option value="editor">editor</option>
          </select>
        </label>
        <label className="field">
          <span>Expires at</span>
          <input
            onChange={(event) => setExpiresAt(event.target.value)}
            type="datetime-local"
            value={expiresAt}
          />
        </label>
        <button className="primary-button permission-submit" type="submit">
          Create
        </button>
      </form>

      {created ? (
        <section className="share-token-panel">
          <p className="document-meta">Token is shown once.</p>
          <div className="share-copy-row">
            <input readOnly value={created.url} />
            <button
              className="secondary-button"
              onClick={() => {
                void navigator.clipboard.writeText(created.url).then(() => setCopied(true));
              }}
              type="button"
            >
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
        </section>
      ) : null}
      {actionError ? <p className="form-error">{actionError}</p> : null}
      {state.status === 'loading' ? <section className="empty-state">Loading</section> : null}
      {state.status === 'error' ? <section className="empty-state">{state.error}</section> : null}
      {state.status === 'success' && state.items.length === 0 ? (
        <section className="empty-state">No share links yet.</section>
      ) : null}
      {state.status === 'success' && state.items.length > 0 ? (
        <section className="document-list">
          {state.items.map((item) => (
            <article className="document-row share-row" key={item.id}>
              <span className="document-title">{item.permission}</span>
              <span className="document-meta">{item.disabled ? 'disabled' : 'active'}</span>
              <span className="document-meta">{item.expiresAt ? formatDate(item.expiresAt) : 'No expiry'}</span>
              <span className="document-meta">{formatDate(item.createdAt)}</span>
              <button
                className="secondary-button danger-button"
                disabled={item.disabled}
                onClick={() => void onDisable(item.id)}
                type="button"
              >
                Disable
              </button>
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
