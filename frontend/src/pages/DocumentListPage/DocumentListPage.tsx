import { ChangeEvent, useEffect, useState } from 'react';

import {
  deleteDocument,
  documentDownloadURL,
  documentOpenURL,
  listDocuments,
  uploadDocument,
  type DocumentItem,
} from '../../api/documents';
import { errorMessage } from '../../api/client';
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
        error: errorMessage(error, '加载文档失败'),
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
      setActionError(errorMessage(error, '上传失败'));
    } finally {
      setUploading(false);
    }
  }

  async function onDelete(id: string) {
    if (!window.confirm('确认删除这个文档？')) {
      return;
    }
    setActionError(null);
    try {
      await deleteDocument(id);
      await refreshDocuments();
    } catch (error) {
      setActionError(errorMessage(error, '删除失败'));
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

  return (
    <main className="app-shell">
      <header className="topbar workspace-header">
        <div className="workspace-intro">
          <p className="eyebrow">DOCS COLLAB / 工作区</p>
          <h1>我的文档</h1>
          <p className="workspace-description">集中管理、协作与沉淀每一份工作内容。</p>
        </div>
        <div className="user-actions">
          {auth.user.isAdmin ? (
            <a className="secondary-button" href="/admin">
              管理控制台
            </a>
          ) : null}
          <label className="upload-button">
            <input
              accept=".docx,.xlsx,.md"
              disabled={uploading}
              onChange={onUpload}
              type="file"
            />
            {uploading ? '上传中' : '上传'}
          </label>
          <a className="secondary-button" href="/profile">
            {auth.user.displayName}
          </a>
          <button className="secondary-button" onClick={() => void auth.logout()} type="button">
            退出登录
          </button>
        </div>
      </header>
      {actionError ? <p className="form-error">{actionError}</p> : null}
      <section className="workspace-content" aria-label="文档列表">
        <div className="workspace-section-heading">
          <div>
            <p className="section-kicker">DOCUMENT LIBRARY</p>
            <h2>最近更新</h2>
          </div>
          {documents.status === 'success' ? (
            <p className="document-count">共 {documents.items.length} 份文档</p>
          ) : null}
        </div>
        {documents.status === 'loading' ? <section className="empty-state">正在整理你的文档…</section> : null}
        {documents.status === 'error' ? (
          <section className="empty-state">{documents.error}</section>
        ) : null}
        {documents.status === 'success' && documents.items.length === 0 ? (
          <section className="empty-state">还没有文档。使用右上角的“上传”开始创建你的工作区。</section>
        ) : null}
        {documents.status === 'success' && documents.items.length > 0 ? (
          <section className="document-list">
            {documents.items.map((document) => (
              <article className="document-row" key={document.id}>
                <a className="document-title" href={`/documents/${document.id}`}>
                  {document.title}
                </a>
                <span className="file-badge">{document.fileExt}</span>
                <span className="document-meta">{formatDate(document.updatedAt)}</span>
                <span className="document-meta">{formatSize(document.sizeBytes)}</span>
                <div className="document-actions">
                  <a className="primary-link compact-link" href={documentOpenURL(document)}>
                    打开
                  </a>
                  <a className="secondary-button" href={documentDownloadURL(document.id)}>
                    下载
                  </a>
                  {document.canManage ? (
                    <a className="secondary-button" href={`/documents/${document.id}/permissions`}>
                      权限
                    </a>
                  ) : null}
                  {document.canManage ? (
                    <a className="secondary-button" href={`/documents/${document.id}/share`}>
                      分享
                    </a>
                  ) : null}
                  {document.canManage ? (
                    <button
                      className="secondary-button danger-button"
                      onClick={() => void onDelete(document.id)}
                      type="button"
                    >
                      删除
                    </button>
                  ) : null}
                </div>
              </article>
            ))}
          </section>
        ) : null}
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
