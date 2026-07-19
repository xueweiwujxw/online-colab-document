import { useEffect, useState } from 'react';

import {
  documentVersionDownloadURL,
  getDocument,
  listDocumentVersions,
  restoreDocumentVersion,
  type DocumentItem,
  type DocumentVersion,
} from '../../api/documents';
import { errorMessage } from '../../api/client';
import { RequireAuth } from '../../components/PageState';

type VersionState =
  | { status: 'loading'; document: null; items: DocumentVersion[]; error: null }
  | { status: 'success'; document: DocumentItem; items: DocumentVersion[]; error: null }
  | { status: 'error'; document: null; items: DocumentVersion[]; error: string };

export function VersionPage({ documentId }: { documentId: string }) {
  const [state, setState] = useState<VersionState>({
    status: 'loading',
    document: null,
    items: [],
    error: null,
  });
  const [actionError, setActionError] = useState<string | null>(null);
  const [restoringVersionId, setRestoringVersionId] = useState<string | null>(null);

  async function refresh() {
    setState((current) => ({ status: 'loading', document: current.document, items: current.items, error: null }) as VersionState);
    try {
      const [document, items] = await Promise.all([
        getDocument(documentId),
        listDocumentVersions(documentId),
      ]);
      setState({ status: 'success', document, items, error: null });
    } catch (error) {
      setState({
        status: 'error',
        document: null,
        items: [],
        error: errorMessage(error, 'Failed to load versions'),
      });
    }
  }

  useEffect(() => {
    void refresh();
  }, [documentId]);

  async function onRestore(version: DocumentVersion) {
    if (!window.confirm(`Restore version ${version.versionNo}?`)) {
      return;
    }
    setActionError(null);
    setRestoringVersionId(version.id);
    try {
      await restoreDocumentVersion(documentId, version.id);
      await refresh();
    } catch (error) {
      setActionError(errorMessage(error, 'Restore failed'));
    } finally {
      setRestoringVersionId(null);
    }
  }

  return (
    <RequireAuth>
      <main className="app-shell">
        <a className="back-link" href={`/documents/${documentId}`}>
          Back to document
        </a>
        <header className="detail-header">
          <div>
            <p className="eyebrow">Versions</p>
            <h1>{state.document?.title ?? 'Document history'}</h1>
          </div>
        </header>
        {actionError ? <p className="form-error">{actionError}</p> : null}
        {state.status === 'loading' ? <section className="empty-state">Loading</section> : null}
        {state.status === 'error' ? <section className="empty-state">{state.error}</section> : null}
        {state.status === 'success' && state.items.length === 0 ? (
          <section className="empty-state">No versions yet.</section>
        ) : null}
        {state.status === 'success' && state.items.length > 0 ? (
          <section className="version-list">
            {state.items.map((version) => (
              <article className="version-row" key={version.id}>
                <span>Version {version.versionNo}</span>
                <span>{version.createdBy ?? 'system'}</span>
                <span>{formatSize(version.sizeBytes)}</span>
                <span>{formatDate(version.createdAt)}</span>
                <div className="version-actions">
                  <a
                    className="secondary-button"
                    href={documentVersionDownloadURL(documentId, version.id)}
                  >
                    Download
                  </a>
                  {state.document.canEdit ? (
                    <button
                      className="secondary-button"
                      disabled={restoringVersionId === version.id}
                      onClick={() => void onRestore(version)}
                      type="button"
                    >
                      {restoringVersionId === version.id ? 'Restoring' : 'Restore'}
                    </button>
                  ) : null}
                </div>
              </article>
            ))}
          </section>
        ) : null}
      </main>
    </RequireAuth>
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
