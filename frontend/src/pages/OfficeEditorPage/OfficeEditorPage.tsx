import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { EmbedHostTransport, type SaveRequestData } from '@casualoffice/sheets/embed';

import {
  fetchOfficeContent,
  getOfficeSession,
  saveOfficeContent,
  type OfficeSession,
} from '../../api/office';
import { errorMessage } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';

type EditorState =
  | { status: 'loading'; session: null; buffer: null; error: null }
  | { status: 'success'; session: OfficeSession; buffer: ArrayBuffer; error: null }
  | { status: 'error'; session: null; buffer: null; error: string };

type SaveState = 'idle' | 'saving' | 'saved' | 'error';

export function OfficeEditorPage({ documentId }: { documentId: string }) {
  const auth = useAuth();
  const activeTransport = useRef<EmbedHostTransport | null>(null);
  const [state, setState] = useState<EditorState>({
    status: 'loading',
    session: null,
    buffer: null,
    error: null,
  });
  const [saveState, setSaveState] = useState<SaveState>('idle');
  const [saveError, setSaveError] = useState<string | null>(null);
  const onTransport = useCallback((transport: EmbedHostTransport | null) => {
    activeTransport.current = transport;
  }, []);
  const onSaveStart = useCallback(() => {
    setSaveState('saving');
    setSaveError(null);
  }, []);
  const onSaveSuccess = useCallback(() => setSaveState('saved'), []);
  const onSaveError = useCallback((message: string) => {
    setSaveState('error');
    setSaveError(message);
  }, []);

  useEffect(() => {
    if (auth.status !== 'authenticated') {
      return;
    }
    let mounted = true;
    setState({ status: 'loading', session: null, buffer: null, error: null });
    getOfficeSession(documentId)
      .then(async (session) => {
        const buffer = await fetchOfficeContent(session.downloadUrl);
        if (mounted) {
          setState({ status: 'success', session, buffer, error: null });
        }
      })
      .catch((error: unknown) => {
        if (mounted) {
          setState({
            status: 'error',
            session: null,
            buffer: null,
            error: errorMessage(error, '加载编辑器失败'),
          });
        }
      });
    return () => {
      mounted = false;
    };
  }, [auth.status, documentId]);

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
    <main className="editor-shell">
      <header className="editor-topbar">
        <a className="back-link" href={`/documents/${documentId}`}>
          返回文档
        </a>
        <div className="office-toolbar">
          {state.status === 'success' ? (
            <span className="connection-pill">{state.session.mode === 'edit' ? '可编辑' : '只读'}</span>
          ) : null}
          {saveError ? <span className="form-error office-save-error">{saveError}</span> : null}
          {state.status === 'success' && state.session.mode === 'edit' ? (
            <button
              className="secondary-button"
              disabled={saveState === 'saving'}
              onClick={() => {
                setSaveState('saving');
                setSaveError(null);
                activeTransport.current?.sendCommandSave();
              }}
              type="button"
            >
              {saveState === 'saving' ? '保存中' : saveState === 'saved' ? '已保存' : '保存'}
            </button>
          ) : null}
        </div>
      </header>
      {state.status === 'loading' ? <section className="empty-state editor-state">加载中</section> : null}
      {state.status === 'error' ? <section className="empty-state editor-state">{state.error}</section> : null}
      {state.status === 'success' ? (
        <CasualIframeHost
          buffer={state.buffer}
          onSaveError={onSaveError}
          onSaveStart={onSaveStart}
          onSaveSuccess={onSaveSuccess}
          onTransport={onTransport}
          session={state.session}
        />
      ) : null}
    </main>
  );
}

function CasualIframeHost({
  session,
  buffer,
  onTransport,
  onSaveStart,
  onSaveSuccess,
  onSaveError,
}: {
  session: OfficeSession;
  buffer: ArrayBuffer;
  onTransport: (transport: EmbedHostTransport | null) => void;
  onSaveStart: () => void;
  onSaveSuccess: () => void;
  onSaveError: (message: string) => void;
}) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const latestBuffer = useMemo(() => buffer.slice(0), [buffer]);
  const embedOrigin = window.location.origin;
  const app = session.fileExt === 'xlsx' ? 'sheet' : 'docs';
  const viewMode = session.mode === 'edit' ? 'editor' : 'preview';

  useEffect(() => {
    const iframe = iframeRef.current;
    if (!iframe?.contentWindow) {
      return;
    }
    if (session.fileExt !== 'docx' && session.fileExt !== 'xlsx') {
      return;
    }
    const transport = new EmbedHostTransport({
      app,
      iframeWindow: iframe.contentWindow,
      embedOrigin,
    });
    onTransport(transport);
    transport.on({
      onEditorReady: () => {
        transport.sendHostHello({ capabilities: ['load', 'save'] });
        transport.sendSetLocale({ locale: 'zh-CN' });
        transport.sendSetTheme({ theme: 'light' });
        transport.sendSetViewMode({ viewMode });
        transport.sendSetReadOnly({ readOnly: session.mode !== 'edit' });
      },
      onLoadRequest: () => ({
        ok: true,
        bytes: latestBuffer.slice(0),
        fileName: session.title,
        readOnly: session.mode !== 'edit',
      }),
      onSaveRequest: async (request: SaveRequestData) => {
        if (session.mode !== 'edit') {
          return { ok: false, code: 'readonly', message: '只读模式不能保存' };
        }
        onSaveStart();
        const result = await saveOfficeContent(session.saveUrl, request.bytes);
        onSaveSuccess();
        return { ok: true, etag: result.etag };
      },
      onError: (error: { message: string }) => {
        onSaveError(error.message);
      },
    });
    const helloTimer = window.setTimeout(() => {
      transport.sendHostHello({ capabilities: ['load', 'save'] });
    }, 100);
    return () => {
      window.clearTimeout(helloTimer);
      transport.destroy();
      onTransport(null);
    };
  }, [app, embedOrigin, latestBuffer, onSaveError, onSaveStart, onSaveSuccess, onTransport, session, viewMode]);

  if (session.fileExt !== 'docx' && session.fileExt !== 'xlsx') {
    return <section className="empty-state editor-state">当前 POC 仅支持 docx / xlsx。</section>;
  }

  return (
    <section className="office-frame">
      <iframe
        className="office-sheet-iframe"
        ref={iframeRef}
        src={`${embedBasePath(session.fileExt)}/embed.html?app=${app}&docId=${encodeURIComponent(
          session.documentId,
        )}&viewMode=${viewMode}`}
        title={session.title}
      />
    </section>
  );
}

function embedBasePath(fileExt: string): string {
  return fileExt === 'xlsx' ? '/casual-sheets' : '/casual-docs';
}
