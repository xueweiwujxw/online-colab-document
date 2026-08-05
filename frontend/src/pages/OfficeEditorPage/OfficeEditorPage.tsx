import { useEffect, useState } from 'react';

import { ApiError, errorMessage } from '../../api/client';
import { getOfficeSession } from '../../api/office';
import { useAuth } from '../../auth/AuthContext';

/**
 * The actual Office UI lives in the official Casual Docker services. Keeping
 * this small hand-off page in the app means the session is always minted by
 * our server-side permission service, rather than embedding a public editor
 * URL or handing the browser storage credentials.
 */
export function OfficeEditorPage({ documentId }: { documentId: string }) {
  const auth = useAuth();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (auth.status !== 'authenticated') {
      return;
    }
    let active = true;
    getOfficeSession(documentId)
      .then((session) => {
        if (active) {
          window.location.replace(session.editorUrl);
        }
      })
      .catch((cause: unknown) => {
        if (!active) {
          return;
        }
        setError(
          cause instanceof ApiError && cause.status === 400
            ? '此文件类型尚未通过 Casual Office 的兼容性验证。'
            : errorMessage(cause, '无法创建 Casual Office 编辑会话'),
        );
      });
    return () => {
      active = false;
    };
  }, [auth.status, documentId]);

  if (auth.status === 'anonymous') {
    window.location.replace('/login');
    return null;
  }

  return (
    <main className="app-shell">
      <section className="empty-state">
        {error ?? '正在安全地打开 Casual Office 编辑器…'}
        {error ? <a className="back-link" href={`/documents/${documentId}`}>返回文档</a> : null}
      </section>
    </main>
  );
}
