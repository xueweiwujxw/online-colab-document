import { ChangeEvent, useEffect, useState } from 'react';

import {
  deleteDocument,
  documentDownloadURL,
  listDocuments,
  uploadDocument,
  type DocumentItem,
} from '../../api/documents';
import { useAuth } from '../../auth/AuthContext';

type DocumentsState =
  | { status: 'loading'; items: DocumentItem[]; error: null }
  | { status: 'success'; items: DocumentItem[]; error: null }
  | { status: 'error'; items: DocumentItem[]; error: string };

export function DocumentListPage() {
  const auth = useAuth();
  const [documents, setDocuments] = useState<DocumentsState>({
    status: 'loading',
    items: [],
    error: null,
  });
  const [uploading, setUploading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  async function refreshDocuments() {
    setDocuments((current) => ({ status: 'loading', items: current.items, error: null }));
    try {
      const items = await listDocuments();
      setDocuments({ status: 'success', items, error: null });
    } catch (error) {
      setDocuments({
        status: 'error',
        items: [],
        error: error instanceof Error ? error.message : 'Failed to load documents',
      });
    }
  }

  useEffect(() => {
    if (auth.status === 'authenticated') {
      void refreshDocuments();
    }
  }, [auth.status]);

  async function onUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) {
      return;
    }
    setUploading(true);
    setActionError(null);
    try {
      await uploadDocument(file);
      await refreshDocuments();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : 'Upload failed');
    } finally {
      setUploading(false);
    }
  }

  async function onDelete(id: string) {
    setActionError(null);
    try {
      await deleteDocument(id);
      await refreshDocuments();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : 'Delete failed');
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

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Documents</p>
          <h1>My documents</h1>
        </div>
        <div className="user-actions">
          {auth.user.isAdmin ? (
            <a className="secondary-button" href="/admin/audit-logs">
              Audit logs
            </a>
          ) : null}
          <label className="upload-button">
            <input disabled={uploading} onChange={onUpload} type="file" />
            {uploading ? 'Uploading' : 'Upload'}
          </label>
          <span>{auth.user.displayName}</span>
          <button className="secondary-button" onClick={() => void auth.logout()} type="button">
            Sign out
          </button>
        </div>
      </header>
      {actionError ? <p className="form-error">{actionError}</p> : null}
      {documents.status === 'loading' ? <section className="empty-state">Loading</section> : null}
      {documents.status === 'error' ? (
        <section className="empty-state">{documents.error}</section>
      ) : null}
      {documents.status === 'success' && documents.items.length === 0 ? (
        <section className="empty-state">No documents yet.</section>
      ) : null}
      {documents.status === 'success' && documents.items.length > 0 ? (
        <section className="document-list">
          {documents.items.map((document) => (
            <article className="document-row" key={document.id}>
              <a className="document-title" href={`/documents/${document.id}`}>
                {document.title}
              </a>
              <span className="document-meta">{document.fileExt}</span>
              <span className="document-meta">{formatDate(document.updatedAt)}</span>
              <span className="document-meta">{formatSize(document.sizeBytes)}</span>
              <div className="document-actions">
                <a className="secondary-button" href={documentDownloadURL(document.id)}>
                  Download
                </a>
                {document.canManage ? (
                  <button
                    className="secondary-button danger-button"
                    onClick={() => void onDelete(document.id)}
                    type="button"
                  >
                    Delete
                  </button>
                ) : null}
              </div>
            </article>
          ))}
        </section>
      ) : null}
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
