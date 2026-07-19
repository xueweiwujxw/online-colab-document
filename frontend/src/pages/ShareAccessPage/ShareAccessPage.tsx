import { useEffect, useMemo, useState } from 'react';

import {
  getShareAccess,
  saveSharedMarkdown,
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
  const [draft, setDraft] = useState('');
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [saveError, setSaveError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    getShareAccess(token)
      .then((access) => {
        if (mounted) {
          setState({ status: 'success', access, error: null });
          setDraft(access.content ?? '');
        }
      })
      .catch((error: unknown) => {
        if (mounted) {
          setState({
            status: 'error',
            access: null,
            error: errorMessage(error, 'Failed to load share link'),
          });
        }
      });
    return () => {
      mounted = false;
    };
  }, [token]);

  const preview = useMemo(() => renderMarkdownPreview(draft), [draft]);

  async function onSave() {
    if (state.status !== 'success' || !state.access.canEdit || saveState === 'saving') {
      return;
    }
    setSaveState('saving');
    setSaveError(null);
    try {
      const access = await saveSharedMarkdown(token, draft);
      setState({ status: 'success', access, error: null });
      setDraft(access.content ?? '');
      setSaveState('saved');
    } catch (error) {
      setSaveState('error');
      setSaveError(errorMessage(error, 'Save failed'));
    }
  }

  if (state.status === 'loading') {
    return (
      <main className="app-shell">
        <section className="empty-state">Loading</section>
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

  const isMarkdown = ['md', 'markdown'].includes(state.access.document.fileExt.toLowerCase());
  const dirty = draft !== (state.access.content ?? '');

  return (
    <main className="editor-shell markdown-shell">
      <header className="editor-topbar">
        <div>
          <p className="eyebrow">Shared document</p>
          <strong>{state.access.document.title}</strong>
        </div>
        <div className="markdown-toolbar">
          <span className="document-meta">{state.access.canEdit ? 'Editable link' : 'Read only link'}</span>
          <a className="secondary-button" href={sharedDownloadURL(token)}>
            Download
          </a>
          {isMarkdown ? (
            <button
              className="primary-button markdown-save-button"
              disabled={!state.access.canEdit || !dirty || saveState === 'saving'}
              onClick={() => void onSave()}
              type="button"
            >
              {saveState === 'saving' ? 'Saving' : 'Save'}
            </button>
          ) : null}
        </div>
      </header>
      {saveError ? <p className="form-error markdown-save-error">{saveError}</p> : null}
      {isMarkdown ? (
        <section className="markdown-editor-grid">
          <label className="markdown-pane">
            <span>Markdown</span>
            <textarea
              readOnly={!state.access.canEdit}
              onChange={(event) => {
                setDraft(event.target.value);
                setSaveState('idle');
              }}
              value={draft}
            />
          </label>
          <section className="markdown-pane markdown-preview-pane">
            <span>Preview</span>
            <div className="markdown-preview">{preview}</div>
          </section>
        </section>
      ) : (
        <section className="empty-state">Use download to open this shared document.</section>
      )}
    </main>
  );
}

function renderMarkdownPreview(content: string) {
  if (content.trim() === '') {
    return <p className="empty-inline">Nothing to preview.</p>;
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
