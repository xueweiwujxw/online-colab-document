import { FormEvent, useEffect, useState } from 'react';

import {
  deletePermission,
  grantPermission,
  listPermissions,
  type DocumentPermission,
} from '../../api/permissions';
import { errorMessage } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';
import { searchUsers, type UserSearchItem } from '../../api/users';

type PermissionState =
  | { status: 'loading'; items: DocumentPermission[]; error: null }
  | { status: 'success'; items: DocumentPermission[]; error: null }
  | { status: 'error'; items: DocumentPermission[]; error: string };

type UserSearchState =
  | { status: 'idle' | 'loading'; items: UserSearchItem[]; hasMore: boolean; error: null }
  | { status: 'error'; items: UserSearchItem[]; hasMore: boolean; error: string };

export function PermissionPage({ documentId }: { documentId: string }) {
  const auth = useAuth();
  const [state, setState] = useState<PermissionState>({
    status: 'loading',
    items: [],
    error: null,
  });
  const [subjectId, setSubjectId] = useState('');
  const [selectedUser, setSelectedUser] = useState<UserSearchItem | null>(null);
  const [userQuery, setUserQuery] = useState('');
  const [userSearch, setUserSearch] = useState<UserSearchState>({
    status: 'idle',
    items: [],
    hasMore: false,
    error: null,
  });
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
      void refreshUsers('');
    }
  }, [auth.status, documentId]);

  async function refreshUsers(query = userQuery, offset = 0) {
    setUserSearch((current) => ({
      status: 'loading',
      items: offset === 0 ? current.items : current.items,
      hasMore: current.hasMore,
      error: null,
    }));
    try {
      const result = await searchUsers(query, offset);
      setUserSearch((current) => ({
        status: 'idle',
        items: offset === 0 ? result.items : mergeUsers(current.items, result.items),
        hasMore: result.hasMore,
        error: null,
      }));
    } catch (error) {
      setUserSearch({
        status: 'error',
        items: offset === 0 ? [] : userSearch.items,
        hasMore: false,
        error: errorMessage(error, '加载用户失败'),
      });
    }
  }

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
      setSelectedUser(null);
      await refreshPermissions();
      await refreshUsers();
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
      await refreshUsers();
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

  const currentUserId = auth.user.id;
  const visibleUsers = userSearch.items.filter((item) => item.id !== currentUserId);

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
          <span>搜索用户</span>
          <input
            onChange={(event) => setUserQuery(event.target.value)}
            placeholder="邮箱或显示名"
            type="text"
            value={userQuery}
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
        <button
          className="secondary-button permission-submit"
          onClick={() => void refreshUsers()}
          type="button"
        >
          搜索
        </button>
        <button className="primary-button permission-submit" disabled={subjectId === ''} type="submit">
          授权
        </button>
      </form>

      {actionError ? <p className="form-error">{actionError}</p> : null}
      {selectedUser ? (
        <p className="document-meta selected-user">
          已选择：{selectedUser.displayName}（{selectedUser.email}）
        </p>
      ) : (
        <p className="document-meta selected-user">请选择一个用户后授权。</p>
      )}
      <section className="user-picker">
        {userSearch.status === 'loading' ? <span className="empty-inline">加载用户中</span> : null}
        {userSearch.status === 'error' ? <span className="form-error">{userSearch.error}</span> : null}
        {userSearch.status !== 'loading' && visibleUsers.length === 0 ? (
          <span className="empty-inline">暂无匹配用户。</span>
        ) : null}
        {visibleUsers.map((item) => {
          const existingPermission = permissionForUser(state.items, item.id);
          return (
            <button
              className={`user-result ${item.id === subjectId ? 'user-result-selected' : ''}`}
              key={item.id}
              onClick={() => {
                setSubjectId(item.id);
                setSelectedUser(item);
                if (existingPermission === 'viewer' || existingPermission === 'editor') {
                  setPermission(existingPermission);
                }
              }}
              type="button"
            >
              <strong>{item.displayName}</strong>
              <span>{item.email}</span>
              <small className={existingPermission ? 'user-permission-badge' : 'user-permission-empty'}>
                {existingPermission ? `已有：${permissionLabel(existingPermission)}` : '未授权'}
              </small>
            </button>
          );
        })}
      </section>
      {userSearch.hasMore ? (
        <button
          className="secondary-button"
          disabled={userSearch.status === 'loading'}
          onClick={() => void refreshUsers(userQuery, userSearch.items.length)}
          type="button"
        >
          {userSearch.status === 'loading' ? '加载中' : '加载更多用户'}
        </button>
      ) : null}
      {state.status === 'loading' ? <section className="empty-state">加载中</section> : null}
      {state.status === 'error' ? <section className="empty-state">{state.error}</section> : null}
      {state.status === 'success' && state.items.length === 0 ? (
        <section className="empty-state">暂无权限。</section>
      ) : null}
      {state.status === 'success' && state.items.length > 0 ? (
        <section className="document-list">
          {state.items.map((item) => (
            <article className="document-row permission-row" key={item.id}>
              <span className="document-title">{permissionSubjectLabel(item)}</span>
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

function permissionSubjectLabel(item: DocumentPermission): string {
  if (item.subjectDisplayName && item.subjectEmail) {
    return `${item.subjectDisplayName}（${item.subjectEmail}）`;
  }
  return item.subjectDisplayName ?? item.subjectEmail ?? item.subjectId;
}

function permissionForUser(items: DocumentPermission[], userId: string): string | null {
  return items.find((item) => item.subjectType === 'user' && item.subjectId === userId)?.permission ?? null;
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

function mergeUsers(current: UserSearchItem[], next: UserSearchItem[]): UserSearchItem[] {
  const items = new Map(current.map((item) => [item.id, item]));
  next.forEach((item) => items.set(item.id, item));
  return [...items.values()];
}
