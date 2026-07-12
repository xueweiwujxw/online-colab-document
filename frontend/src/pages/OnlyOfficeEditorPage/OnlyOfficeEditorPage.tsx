import { useEffect, useRef, useState } from 'react';

import { getOnlyOfficeConfig, type OnlyOfficeConfig } from '../../api/onlyoffice';
import { useAuth } from '../../auth/AuthContext';

declare global {
  interface Window {
    DocsAPI?: {
      DocEditor: new (elementId: string, config: OnlyOfficeConfig) => unknown;
    };
  }
}

type EditorState =
  | { status: 'loading'; config: null; error: null }
  | { status: 'success'; config: OnlyOfficeConfig; error: null }
  | { status: 'error'; config: null; error: string };

export function OnlyOfficeEditorPage({ documentId }: { documentId: string }) {
  const auth = useAuth();
  const editorRef = useRef<unknown>(null);
  const [state, setState] = useState<EditorState>({
    status: 'loading',
    config: null,
    error: null,
  });

  useEffect(() => {
    if (auth.status !== 'authenticated') {
      return;
    }
    let mounted = true;
    getOnlyOfficeConfig(documentId)
      .then((config) => {
        if (mounted) {
          setState({ status: 'success', config, error: null });
        }
      })
      .catch((error: unknown) => {
        if (mounted) {
          setState({
            status: 'error',
            config: null,
            error: error instanceof Error ? error.message : 'Failed to load editor',
          });
        }
      });
    return () => {
      mounted = false;
    };
  }, [auth.status, documentId]);

  useEffect(() => {
    if (state.status !== 'success') {
      return;
    }
    const scriptSrc = `${state.config.documentServerUrl.replace(/\/$/, '')}/web-apps/apps/api/documents/api.js`;
    loadScript(scriptSrc)
      .then(() => {
        if (window.DocsAPI) {
          editorRef.current = new window.DocsAPI.DocEditor('onlyoffice-editor', state.config);
        }
      })
      .catch(() => {
        setState({ status: 'error', config: null, error: 'Failed to load ONLYOFFICE editor' });
      });
  }, [state]);

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
    <main className="editor-shell">
      <header className="editor-topbar">
        <a className="back-link" href={`/documents/${documentId}`}>
          Back to document
        </a>
        {state.status === 'success' ? (
          <span className="document-meta">{state.config.editorConfig.mode}</span>
        ) : null}
      </header>
      {state.status === 'loading' ? <section className="empty-state">Loading</section> : null}
      {state.status === 'error' ? <section className="empty-state">{state.error}</section> : null}
      <section className="onlyoffice-frame" id="onlyoffice-editor" />
    </main>
  );
}

function loadScript(src: string): Promise<void> {
  const existing = document.querySelector<HTMLScriptElement>(`script[src="${src}"]`);
  if (existing) {
    return Promise.resolve();
  }
  return new Promise((resolve, reject) => {
    const script = document.createElement('script');
    script.src = src;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error('script failed'));
    document.head.appendChild(script);
  });
}
