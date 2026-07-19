import { FormEvent, useEffect, useState } from 'react';

import {
  deletePermission,
  grantPermission,
  listPermissions,
  type DocumentPermission,
} from '../../api/permissions';
import { errorMessage } from '../../api/client';
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
        error: errorMessage(error, '加载权限失败'),
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
      setActionError(errorMessage(error, '授权失败'));
    }
  }

  async function onDelete(permissionId: string) {
    if (!window.confirm('确认删除这个权限？')) {
      return;
    }
    setActionError(null);
    try {
      await deletePermission(documentId, permissionId);
      await refreshPermissions();
    } catch (error) {
      setActionError(errorMessage(error, '删除失败'));
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
          <p className="eyebrow">权限</p>
          <h1>管理访问</h1>
        </div>
      </header>

      <form className="permission-form" onSubmit={onSubmit}>
        <label className="field">
          <span>用户 ID</span>
          <input
            onChange={(event) => setSubjectId(event.target.value)}
            required
            type="text"
            value={subjectId}
          />
        </label>
        <label className="field">
          <span>权限</span>
          <select
            onChange={(event) => setPermission(event.target.value as 'viewer' | 'editor')}
            value={permission}
          >
            <option value="viewer">只读</option>
            <option value="editor">可编辑</option>
          </select>
        </label>
        <button className="primary-button permission-submit" type="submit">
          授权
        </button>
      </form>

      {actionError ? <p className="form-error">{actionError}</p> : null}
      {state.status === 'loading' ? <section className="empty-state">加载中</section> : null}
      {state.status === 'error' ? <section className="empty-state">{state.error}</section> : null}
      {state.status === 'success' && state.items.length === 0 ? (
        <section className="empty-state">暂无权限。</section>
      ) : null}
      {state.status === 'success' && state.items.length > 0 ? (
        <section className="document-list">
          {state.items.map((item) => (
            <article className="document-row permission-row" key={item.id}>
              <span className="document-title">{item.subjectId}</span>
              <span className="document-meta">{subjectTypeLabel(item.subjectType)}</span>
              <span className="document-meta">{permissionLabel(item.permission)}</span>
              <button
                className="secondary-button danger-button"
                onClick={() => void onDelete(item.id)}
                type="button"
              >
                删除
              </button>
            </article>
          ))}
        </section>
      ) : null}
    </main>
  );
}

function permissionLabel(permission: string): string {
  if (permission === 'owner') {
    return '所有者';
  }
  if (permission === 'editor') {
    return '可编辑';
  }
  if (permission === 'viewer') {
    return '只读';
  }
  return permission;
}

function subjectTypeLabel(subjectType: string): string {
  if (subjectType === 'user') {
    return '用户';
  }
  return subjectType;
}
