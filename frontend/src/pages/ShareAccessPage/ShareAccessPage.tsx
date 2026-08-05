import { useEffect, useState } from 'react';

import {
  getShareAccess,
  sharedDownloadURL,
  type ShareAccess,
} from '../../api/share';
import { errorMessage } from '../../api/client';

type ShareAccessState =
  | { status: 'loading'; access: null; error: null }
  | { status: 'success'; access: ShareAccess; error: null }
  | { status: 'error'; access: null; error: string };

export function ShareAccessPage({ token }: { token: string }) {
  const [state, setState] = useState<ShareAccessState>({
    status: 'loading',
    access: null,
    error: null,
  });

  useEffect(() => {
    let mounted = true;
    getShareAccess(token)
      .then((access) => {
        if (mounted) {
          setState({ status: 'success', access, error: null });
        }
      })
      .catch((error: unknown) => {
        if (mounted) {
          setState({
            status: 'error',
            access: null,
            error: errorMessage(error, '加载分享链接失败'),
          });
        }
      });
    return () => {
      mounted = false;
    };
  }, [token]);

  if (state.status === 'loading') {
    return (
      <main className="app-shell">
        <section className="empty-state">加载中</section>
      </main>
    );
  }

  if (state.status === 'error') {
    return (
      <main className="app-shell">
        <section className="empty-state">{state.error}</section>
      </main>
    );
  }

  return (
    <main className="editor-shell markdown-shell">
      <header className="editor-topbar">
        <div>
          <p className="eyebrow">分享文档</p>
          <strong>{state.access.document.title}</strong>
        </div>
        <div className="markdown-toolbar">
          <span className="document-meta">{state.access.canEdit ? '可编辑链接' : '只读链接'}</span>
          <a className="secondary-button" href={sharedDownloadURL(token)}>
            下载
          </a>
        </div>
      </header>
      <section className="empty-state">
        分享链接只提供下载；需要在线编辑或协同请使用已登录的 Casual Office 文档入口。
      </section>
    </main>
  );
}
