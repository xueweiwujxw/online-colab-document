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
        error: errorMessage(error, '加载分享链接失败'),
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
      setActionError(errorMessage(error, '创建分享链接失败'));
    }
  }

  async function onDisable(id: string) {
    if (!window.confirm('确认禁用这个分享链接？')) {
      return;
    }
    setActionError(null);
    try {
      await disableShareLink(id);
      await refreshLinks();
    } catch (error) {
      setActionError(errorMessage(error, '禁用分享链接失败'));
    }
  }

  if (auth.status === 'loading') {
    return (
      <main className="app-shell">
        <section className="empty-state">加载中</section>
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
        返回文档
      </a>
      <header className="detail-header">
        <div>
          <p className="eyebrow">分享</p>
          <h1>分享链接</h1>
        </div>
      </header>

      <form className="permission-form" onSubmit={onCreate}>
        <label className="field">
          <span>权限</span>
          <select
            onChange={(event) => setPermission(event.target.value as SharePermission)}
            value={permission}
          >
            <option value="viewer">只读</option>
            <option value="editor">可编辑</option>
          </select>
        </label>
        <label className="field">
          <span>过期时间</span>
          <input
            onChange={(event) => setExpiresAt(event.target.value)}
            type="datetime-local"
            value={expiresAt}
          />
        </label>
        <button className="primary-button permission-submit" type="submit">
          创建
        </button>
      </form>

      {created ? (
        <section className="share-token-panel">
          <p className="document-meta">链接只展示一次，请及时复制。</p>
          <div className="share-copy-row">
            <input readOnly value={created.url} />
            <button
              className="secondary-button"
              onClick={() => {
                void navigator.clipboard.writeText(created.url).then(() => setCopied(true));
              }}
              type="button"
            >
              {copied ? '已复制' : '复制'}
            </button>
          </div>
        </section>
      ) : null}
      {actionError ? <p className="form-error">{actionError}</p> : null}
      {state.status === 'loading' ? <section className="empty-state">加载中</section> : null}
      {state.status === 'error' ? <section className="empty-state">{state.error}</section> : null}
      {state.status === 'success' && state.items.length === 0 ? (
        <section className="empty-state">暂无分享链接。</section>
      ) : null}
      {state.status === 'success' && state.items.length > 0 ? (
        <section className="document-list">
          {state.items.map((item) => (
            <article className="document-row share-row" key={item.id}>
              <span className="document-title">{permissionLabel(item.permission)}</span>
              <span className="document-meta">{item.disabled ? '已禁用' : '启用中'}</span>
              <span className="document-meta">{item.expiresAt ? formatDate(item.expiresAt) : '永不过期'}</span>
              <span className="document-meta">{formatDate(item.createdAt)}</span>
              <button
                className="secondary-button danger-button"
                disabled={item.disabled}
                onClick={() => void onDisable(item.id)}
                type="button"
              >
                禁用
              </button>
            </article>
          ))}
        </section>
      ) : null}
    </main>
  );
}

function permissionLabel(permission: string): string {
  if (permission === 'editor') {
    return '可编辑';
  }
  if (permission === 'viewer') {
    return '只读';
  }
  return permission;
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}
