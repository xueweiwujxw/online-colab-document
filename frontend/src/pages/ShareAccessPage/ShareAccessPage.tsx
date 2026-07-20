import { useEffect, useState } from 'react';

import {
  getShareAccess,
  saveSharedMarkdown,
  sharedDownloadURL,
  type ShareAccess,
} from '../../api/share';
import { errorMessage } from '../../api/client';
import { RichMarkdownEditor } from '../MarkdownEditorPage/MarkdownEditorPage';

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
            error: errorMessage(error, '加载分享链接失败'),
          });
        }
      });
    return () => {
      mounted = false;
    };
  }, [token]);

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
      setSaveError(errorMessage(error, '保存失败'));
    }
  }

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

  const isMarkdown = ['md', 'markdown'].includes(state.access.document.fileExt.toLowerCase());
  const dirty = draft !== (state.access.content ?? '');

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
          {isMarkdown ? (
            <button
              className="primary-button markdown-save-button"
              disabled={!state.access.canEdit || !dirty || saveState === 'saving'}
              onClick={() => void onSave()}
              type="button"
            >
              {saveState === 'saving' ? '保存中' : '保存'}
            </button>
          ) : null}
        </div>
      </header>
      {saveError ? <p className="form-error markdown-save-error">{saveError}</p> : null}
      {isMarkdown ? (
        <section className="markdown-editor-grid rich-markdown-grid">
          <RichMarkdownEditor
            content={draft}
            disabled={false}
            onChange={(content) => {
              setDraft(content);
              setSaveState('idle');
            }}
            readOnly={!state.access.canEdit}
          />
          <section className="markdown-pane markdown-preview-pane">
            <span>Markdown 源码</span>
            <div className="markdown-preview markdown-source-preview">
              <pre>{draft.trim() === '' ? '暂无 Markdown 内容。' : draft}</pre>
            </div>
          </section>
        </section>
      ) : (
        <section className="empty-state">请下载后打开这个分享文档。</section>
      )}
    </main>
  );
}
