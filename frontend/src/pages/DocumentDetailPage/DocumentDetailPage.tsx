import { useEffect, useState } from 'react';

import {
  documentDownloadURL,
  getDocument,
  listDocumentVersions,
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

  useEffect(() => {
    if (auth.status !== 'authenticated') {
      return;
    }
    let mounted = true;
    Promise.all([getDocument(id), listDocumentVersions(id)])
      .then(([document, versions]) => {
        if (mounted) {
          setState({ status: 'success', document, versions, error: null });
        }
      })
      .catch((error: unknown) => {
        if (mounted) {
          setState({
            status: 'error',
            document: null,
            versions: [],
            error: error instanceof Error ? error.message : 'Failed to load document',
          });
        }
      });
    return () => {
      mounted = false;
    };
  }, [auth.status, id]);

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
          {isOfficeDocument(state.document.fileExt) ? (
            <a className="secondary-button" href={`/documents/${state.document.id}/edit`}>
              Open editor
            </a>
          ) : null}
          <a className="primary-link" href={documentDownloadURL(state.document.id)}>
            Download
          </a>
        </div>
      </header>
      <section className="version-list">
        <h2>Versions</h2>
        {state.versions.length === 0 ? (
          <p className="empty-inline">No versions yet.</p>
        ) : (
          state.versions.map((version) => (
            <div className="version-row" key={version.id}>
              <span>Version {version.versionNo}</span>
              <span>{formatSize(version.sizeBytes)}</span>
              <span>{formatDate(version.createdAt)}</span>
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
