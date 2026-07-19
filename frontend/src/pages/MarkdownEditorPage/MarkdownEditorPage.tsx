import { useEffect, useMemo, useState } from 'react';

import {
  documentDownloadURL,
  getMarkdownDocument,
  saveMarkdownDocument,
  type DocumentItem,
} from '../../api/documents';
import { useAuth } from '../../auth/AuthContext';

type MarkdownState =
  | { status: 'loading'; document: null; content: string; canEdit: false; error: null }
  | { status: 'success'; document: DocumentItem; content: string; canEdit: boolean; error: null }
  | { status: 'error'; document: null; content: string; canEdit: false; error: string };

export function MarkdownEditorPage({ documentId }: { documentId: string }) {
  const auth = useAuth();
  const [state, setState] = useState<MarkdownState>({
    status: 'loading',
    document: null,
    content: '',
    canEdit: false,
    error: null,
  });
  const [draft, setDraft] = useState('');
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [saveError, setSaveError] = useState<string | null>(null);

  useEffect(() => {
    if (auth.status !== 'authenticated') {
      return;
    }
    let mounted = true;
    getMarkdownDocument(documentId)
      .then((markdown) => {
        if (mounted) {
          setState({
            status: 'success',
            document: markdown.document,
            content: markdown.content,
            canEdit: markdown.canEdit,
            error: null,
          });
          setDraft(markdown.content);
          setSaveState('idle');
          setSaveError(null);
        }
      })
      .catch((error: unknown) => {
        if (mounted) {
          setState({
            status: 'error',
            document: null,
            content: '',
            canEdit: false,
            error: error instanceof Error ? error.message : 'Failed to load Markdown document',
          });
        }
      });
    return () => {
      mounted = false;
    };
  }, [auth.status, documentId]);

  const preview = useMemo(() => renderMarkdownPreview(draft), [draft]);

  async function onSave() {
    if (state.status !== 'success' || !state.canEdit || saveState === 'saving') {
      return;
    }
    setSaveState('saving');
    setSaveError(null);
    try {
      const markdown = await saveMarkdownDocument(documentId, draft);
      setState({
        status: 'success',
        document: markdown.document,
        content: markdown.content,
        canEdit: markdown.canEdit,
        error: null,
      });
      setDraft(markdown.content);
      setSaveState('saved');
    } catch (error) {
      setSaveState('error');
      setSaveError(error instanceof Error ? error.message : 'Save failed');
    }
  }

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

  if (state.status === 'loading') {
    return (
      <main className="editor-shell markdown-shell">
        <header className="editor-topbar">
          <a className="back-link" href={`/documents/${documentId}`}>
            Back to document
          </a>
        </header>
        <section className="empty-state">Loading</section>
      </main>
    );
  }

  if (state.status === 'error') {
    return (
      <main className="editor-shell markdown-shell">
        <header className="editor-topbar">
          <a className="back-link" href={`/documents/${documentId}`}>
            Back to document
          </a>
        </header>
        <section className="empty-state">{state.error}</section>
      </main>
    );
  }

  const dirty = draft !== state.content;
  const saveLabel = saveState === 'saving' ? 'Saving' : 'Save';

  return (
    <main className="editor-shell markdown-shell">
      <header className="editor-topbar">
        <a className="back-link" href={`/documents/${documentId}`}>
          Back to document
        </a>
        <div className="markdown-toolbar">
          <span className="document-meta">{state.canEdit ? 'Editable' : 'Read only'}</span>
          <span className="document-meta">{saveStatusText(saveState, dirty)}</span>
          <a className="secondary-button" href={documentDownloadURL(documentId)}>
            Download
          </a>
          <button
            className="primary-button markdown-save-button"
            disabled={!state.canEdit || !dirty || saveState === 'saving'}
            onClick={() => void onSave()}
            type="button"
          >
            {saveLabel}
          </button>
        </div>
      </header>
      {saveError ? <p className="form-error markdown-save-error">{saveError}</p> : null}
      <section className="markdown-editor-grid">
        <label className="markdown-pane">
          <span>Markdown</span>
          <textarea
            readOnly={!state.canEdit}
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
    </main>
  );
}

function saveStatusText(saveState: 'idle' | 'saving' | 'saved' | 'error', dirty: boolean): string {
  if (saveState === 'saving') {
    return 'Saving changes';
  }
  if (saveState === 'saved' && !dirty) {
    return 'Saved';
  }
  if (saveState === 'error') {
    return 'Save failed';
  }
  return dirty ? 'Unsaved changes' : 'No changes';
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
