import { useEffect, useMemo, useRef, useState } from 'react';
import * as Y from 'yjs';

import {
  documentDownloadURL,
  getMarkdownSnapshot,
  markdownWebSocketURL,
  type PresenceUser,
} from '../../api/documents';
import { errorMessage } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';

type ConnectionState = 'loading' | 'connected' | 'disconnected' | 'reconnecting' | 'error';

type ServerMessage = {
  type: 'init' | 'update' | 'presence' | 'error';
  content?: string;
  canEdit?: boolean;
  updateSeq?: number;
  users?: PresenceUser[];
  error?: string;
};

export function MarkdownEditorPage({ documentId }: { documentId: string }) {
  const auth = useAuth();
  const docRef = useRef(new Y.Doc());
  const textRef = useRef(docRef.current.getText('markdown'));
  const reconnectRef = useRef<number | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const [draft, setDraft] = useState('');
  const [canEdit, setCanEdit] = useState(false);
  const [users, setUsers] = useState<PresenceUser[]>([]);
  const [connection, setConnection] = useState<ConnectionState>('loading');
  const [error, setError] = useState<string | null>(null);
  const [lastSavedSeq, setLastSavedSeq] = useState<number | null>(null);

  useEffect(() => {
    if (auth.status !== 'authenticated') {
      return;
    }
    let stopped = false;

    function applyContent(content: string) {
      const text = textRef.current;
      docRef.current.transact(() => {
        text.delete(0, text.length);
        text.insert(0, content);
      });
      setDraft(content);
    }

    function connect() {
      setConnection((current) => (current === 'loading' ? 'loading' : 'reconnecting'));
      const socket = new WebSocket(markdownWebSocketURL(documentId));
      socketRef.current = socket;

      socket.onopen = () => {
        if (stopped) {
          socket.close();
          return;
        }
        setConnection('connected');
        setError(null);
        socket.send(JSON.stringify({ type: 'presence' }));
      };

      socket.onmessage = (event) => {
        const message = JSON.parse(event.data as string) as ServerMessage;
        if (message.type === 'init') {
          applyContent(message.content ?? '');
          setCanEdit(Boolean(message.canEdit));
          setUsers(message.users ?? []);
          setLastSavedSeq(message.updateSeq ?? null);
          return;
        }
        if (message.type === 'update') {
          applyContent(message.content ?? '');
          setLastSavedSeq(message.updateSeq ?? null);
          return;
        }
        if (message.type === 'presence') {
          setUsers(message.users ?? []);
          return;
        }
        if (message.type === 'error') {
          setError(message.error ?? '协同编辑错误');
        }
      };

      socket.onerror = () => {
        setConnection('error');
        setError('WebSocket 连接失败');
      };

      socket.onclose = () => {
        if (stopped) {
          return;
        }
        setConnection('disconnected');
        reconnectRef.current = window.setTimeout(connect, 1200);
      };
    }

    getMarkdownSnapshot(documentId)
      .then((snapshot) => {
        if (stopped) {
          return;
        }
        applyContent(snapshot.content);
        setCanEdit(snapshot.canEdit);
        setUsers(snapshot.users);
        setLastSavedSeq(snapshot.versionNo);
        connect();
      })
      .catch((caught: unknown) => {
        if (stopped) {
          return;
        }
        setConnection('error');
        setError(errorMessage(caught, '加载 Markdown 快照失败'));
      });

    return () => {
      stopped = true;
      if (reconnectRef.current !== null) {
        window.clearTimeout(reconnectRef.current);
      }
      socketRef.current?.close();
    };
  }, [auth.status, documentId]);

  const preview = useMemo(() => renderMarkdownPreview(draft), [draft]);

  function onDraftChange(content: string) {
    if (!canEdit) {
      return;
    }
    const text = textRef.current;
    docRef.current.transact(() => {
      text.delete(0, text.length);
      text.insert(0, content);
    });
    setDraft(content);
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify({ type: 'update', content }));
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

  const readonly = !canEdit;
  const statusText = connectionStatusText(connection, readonly);

  return (
    <main className="editor-shell markdown-shell">
      <header className="editor-topbar">
        <a className="back-link" href={`/documents/${documentId}`}>
          返回文档
        </a>
        <div className="markdown-toolbar">
          <span className={`connection-pill connection-${connection}`}>{statusText}</span>
          <span className="document-meta">{lastSavedSeq === null ? '快照' : `序号 ${lastSavedSeq}`}</span>
          <a className="secondary-button" href={documentDownloadURL(documentId)}>
            下载
          </a>
        </div>
      </header>
      {error ? <p className="form-error markdown-save-error">{error}</p> : null}
      <section className="presence-bar">
        {users.length === 0 ? (
          <span className="empty-inline">暂无其他在线用户。</span>
        ) : (
          users.map((user) => (
            <span className="presence-user" key={`${user.userId}-${user.displayName}`}>
              {user.displayName}
              <small>{user.canEdit ? '可编辑' : '只读'}</small>
            </span>
          ))
        )}
      </section>
      <section className="markdown-editor-grid">
        <label className="markdown-pane">
          <span>Markdown 源码</span>
          <textarea
            readOnly={readonly || connection === 'loading'}
            onChange={(event) => onDraftChange(event.target.value)}
            value={draft}
          />
        </label>
        <section className="markdown-pane markdown-preview-pane">
          <span>预览</span>
          <div className="markdown-preview">{preview}</div>
        </section>
      </section>
    </main>
  );
}

function connectionStatusText(connection: ConnectionState, readonly: boolean): string {
  if (connection === 'loading') {
    return '加载中';
  }
  if (connection === 'connected') {
    return readonly ? '已连接，只读' : '已连接';
  }
  if (connection === 'reconnecting') {
    return '重连中';
  }
  if (connection === 'error') {
    return '连接错误';
  }
  return '已断开';
}

function renderMarkdownPreview(content: string) {
  if (content.trim() === '') {
    return <p className="empty-inline">暂无可预览内容。</p>;
  }
  return content.split(/\n{2,}/).map((block, index) => {
    const trimmed = block.trim();
    const heading = trimmed.match(/^(#{1,3})\s+(.+)$/);
    if (heading) {
      const level = heading[1].length;
      const text = heading[2];
      if (level === 1) {
        return <h1 key={index}>{text}</h1>;
      }
      if (level === 2) {
        return <h2 key={index}>{text}</h2>;
      }
      return <h3 key={index}>{text}</h3>;
    }
    if (trimmed.startsWith('- ')) {
      return (
        <ul key={index}>
          {trimmed.split('\n').map((line, itemIndex) => (
            <li key={itemIndex}>{line.replace(/^-\s+/, '')}</li>
          ))}
        </ul>
      );
    }
    if (trimmed.startsWith('```')) {
      return <pre key={index}>{trimmed.replace(/^```\w*\n?/, '').replace(/```$/, '')}</pre>;
    }
    return <p key={index}>{trimmed}</p>;
  });
}
