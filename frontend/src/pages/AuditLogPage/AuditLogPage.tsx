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
        error: errorMessage(error, '加载审计日志失败'),
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
        <section className="empty-state">加载中</section>
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
          返回文档
        </a>
        <section className="empty-state">无权限访问</section>
      </main>
    );
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">管理</p>
          <h1>审计日志</h1>
        </div>
        <div className="user-actions">
          <a className="secondary-button" href="/profile">
            用户中心
          </a>
          <a className="secondary-button" href="/documents">
            文档
          </a>
          <button className="secondary-button" onClick={() => void auth.logout()} type="button">
            退出登录
          </button>
        </div>
      </header>
      <form className="audit-filters" onSubmit={onSubmit}>
        <label className="field">
          操作
          <input
            onChange={(event) => setFilter({ ...filter, action: event.target.value })}
            placeholder="document.upload"
            value={filter.action ?? ''}
          />
        </label>
        <label className="field">
          目标类型
          <input
            onChange={(event) => setFilter({ ...filter, targetType: event.target.value })}
            placeholder="document"
            value={filter.targetType ?? ''}
          />
        </label>
        <label className="field">
          操作者用户 ID
          <input
            onChange={(event) => setFilter({ ...filter, actorUserId: event.target.value })}
            value={filter.actorUserId ?? ''}
          />
        </label>
        <label className="field">
          数量
          <input
            min="1"
            onChange={(event) => setFilter({ ...filter, limit: Number(event.target.value) })}
            type="number"
            value={filter.limit ?? 50}
          />
        </label>
        <button className="secondary-button audit-filter-button" type="submit">
          筛选
        </button>
      </form>
      {state.status === 'loading' ? <section className="empty-state">加载中</section> : null}
      {state.status === 'error' ? <section className="empty-state">{state.error}</section> : null}
      {state.status === 'success' && state.items.length === 0 ? (
        <section className="empty-state">暂无审计日志。</section>
      ) : null}
      {state.status === 'success' && state.items.length > 0 ? (
        <section className="audit-list">
          {state.items.map((item) => (
            <article className="audit-row" key={item.id}>
              <span className="document-meta">{formatDate(item.createdAt)}</span>
              <span className="document-meta">{item.actorUserId ?? '匿名'}</span>
              <strong>{actionLabel(item.action)}</strong>
              <span className="document-meta">
                {targetTypeLabel(item.targetType)}:{item.targetId}
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

function actionLabel(action: string): string {
  const labels: Record<string, string> = {
    'auth.login': '登录',
    'auth.logout': '登出',
    'auth.password_change': '修改密码',
    'document.upload': '上传文档',
    'document.download': '下载文档',
    'document.delete': '删除文档',
    'document.markdown_save': '保存 Markdown',
    'office.save': 'Office 保存',
    'onlyoffice.save': 'ONLYOFFICE 保存',
    'document.version_restore': '恢复版本',
    'permission.grant': '授权',
    'permission.delete': '删除权限',
    'share.create': '创建分享',
    'share.disable': '禁用分享',
    'share.access': '访问分享',
    'share.download': '下载分享',
    'share.markdown_save': '保存分享 Markdown',
  };
  return labels[action] ?? action;
}

function targetTypeLabel(targetType: string): string {
  const labels: Record<string, string> = {
    document: '文档',
    user: '用户',
    share: '分享',
    permission: '权限',
    version: '版本',
  };
  return labels[targetType] ?? targetType;
}
