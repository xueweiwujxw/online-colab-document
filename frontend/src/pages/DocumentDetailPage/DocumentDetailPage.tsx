import { useEffect, useState } from 'react';

import {
  documentDownloadURL,
  documentVersionDownloadURL,
  getDocument,
  listDocumentVersions,
  restoreDocumentVersion,
  type DocumentItem,
  type DocumentVersion,
} from '../../api/documents';
import { useAuth } from '../../auth/AuthContext';

type DetailState =
  | { status: 'loading'; document: null; versions: DocumentVersion[]; error: null }
  | { status: 'success'; document: DocumentItem; versions: DocumentVersion[]; error: null }
  | { status: 'error'; document: null; versions: DocumentVersion[]; error: string };

export function DocumentDetailPage({ id }: { id: string }) {
  const auth = useAuth();
  const [state, setState] = useState<DetailState>({
    status: 'loading',
    document: null,
    versions: [],
    error: null,
  });
  const [actionError, setActionError] = useState<string | null>(null);
  const [restoringVersionId, setRestoringVersionId] = useState<string | null>(null);

  async function refreshDocument(mounted = true) {
    try {
      const [document, versions] = await Promise.all([getDocument(id), listDocumentVersions(id)]);
      if (mounted) {
        setState({ status: 'success', document, versions, error: null });
      }
    } catch (error) {
      if (mounted) {
        setState({
          status: 'error',
          document: null,
          versions: [],
          error: error instanceof Error ? error.message : 'Failed to load document',
        });
      }
    }
  }

  useEffect(() => {
    if (auth.status !== 'authenticated') {
      return;
    }
    let mounted = true;
    void refreshDocument(mounted);
    return () => {
      mounted = false;
    };
  }, [auth.status, id]);

  async function onRestore(versionId: string) {
    setActionError(null);
    setRestoringVersionId(versionId);
    try {
      await restoreDocumentVersion(id, versionId);
      await refreshDocument();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : 'Restore failed');
    } finally {
      setRestoringVersionId(null);
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
      <main className="app-shell">
        <section className="empty-state">Loading</section>
      </main>
    );
  }

  if (state.status === 'error') {
    return (
      <main className="app-shell">
        <a className="back-link" href="/documents">
          Back to documents
        </a>
        <section className="empty-state">{state.error}</section>
      </main>
    );
  }

  return (
    <main className="app-shell">
      <a className="back-link" href="/documents">
        Back to documents
      </a>
      <header className="detail-header">
        <div>
          <p className="eyebrow">{state.document.fileExt}</p>
          <h1>{state.document.title}</h1>
          <p className="document-meta">{state.document.originalFilename}</p>
        </div>
        <div className="detail-actions">
          {state.document.canManage ? (
            <a className="secondary-button" href={`/documents/${state.document.id}/permissions`}>
              Permissions
            </a>
          ) : null}
          {state.document.canManage ? (
            <a className="secondary-button" href={`/documents/${state.document.id}/share`}>
              Share
            </a>
          ) : null}
          {isOfficeDocument(state.document.fileExt) ? (
            <a className="secondary-button" href={`/documents/${state.document.id}/edit`}>
              Open editor
            </a>
          ) : null}
          {isMarkdownDocument(state.document.fileExt) ? (
            <a className="secondary-button" href={`/documents/${state.document.id}/markdown`}>
              Open editor
            </a>
          ) : null}
          <a className="primary-link" href={documentDownloadURL(state.document.id)}>
            Download
          </a>
        </div>
      </header>
      {actionError ? <p className="form-error">{actionError}</p> : null}
      <section className="version-list">
        <h2>Versions</h2>
        {state.versions.length === 0 ? (
          <p className="empty-inline">No versions yet.</p>
        ) : (
          state.versions.map((version) => (
            <div className="version-row" key={version.id}>
              <span>Version {version.versionNo}</span>
              <span>{version.createdBy ?? 'system'}</span>
              <span>{formatSize(version.sizeBytes)}</span>
              <span>{formatDate(version.createdAt)}</span>
              <div className="version-actions">
                <a
                  className="secondary-button"
                  href={documentVersionDownloadURL(state.document.id, version.id)}
                >
                  Download
                </a>
                {state.document.canEdit ? (
                  <button
                    className="secondary-button"
                    disabled={restoringVersionId === version.id}
                    onClick={() => void onRestore(version.id)}
                    type="button"
                  >
                    {restoringVersionId === version.id ? 'Restoring' : 'Restore'}
                  </button>
                ) : null}
              </div>
            </div>
          ))
        )}
      </section>
    </main>
  );
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}

function formatSize(value: number): string {
  if (value < 1024) {
    return `${value} B`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`;
  }
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

function isOfficeDocument(fileExt: string): boolean {
  return ['doc', 'docx', 'xls', 'xlsx'].includes(fileExt.toLowerCase());
}

function isMarkdownDocument(fileExt: string): boolean {
  return ['md', 'markdown'].includes(fileExt.toLowerCase());
}
