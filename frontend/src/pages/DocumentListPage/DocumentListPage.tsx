import { ChangeEvent, DragEvent, useEffect, useMemo, useState } from 'react';

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

type FileTypeFilter = 'all' | 'docx' | 'xlsx' | 'md';
type AccessFilter = 'all' | 'editable' | 'managed';
type SortOrder = 'updated' | 'name' | 'size';

export function DocumentListPage() {
  const auth = useAuth();
  const [documents, setDocuments] = useState<DocumentsState>({
    status: 'loading',
    items: [],
    error: null,
  });
  const [uploading, setUploading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [fileType, setFileType] = useState<FileTypeFilter>('all');
  const [access, setAccess] = useState<AccessFilter>('all');
  const [sortOrder, setSortOrder] = useState<SortOrder>('updated');
  const [isDragging, setIsDragging] = useState(false);

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

  async function uploadFile(file: File) {
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

  async function onUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (file) {
      await uploadFile(file);
    }
  }

  async function onDrop(event: DragEvent<HTMLLabelElement>) {
    event.preventDefault();
    setIsDragging(false);
    const file = event.dataTransfer.files?.[0];
    if (file) {
      await uploadFile(file);
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

  const filteredDocuments = useMemo(() => {
    const normalizedQuery = query.trim().toLocaleLowerCase();
    return documents.items
      .filter((document) => {
        const matchesQuery =
          normalizedQuery === '' ||
          document.title.toLocaleLowerCase().includes(normalizedQuery) ||
          document.originalFilename.toLocaleLowerCase().includes(normalizedQuery);
        const matchesType = fileType === 'all' || document.fileExt.toLowerCase() === fileType;
        const matchesAccess =
          access === 'all' ||
          (access === 'managed' && document.canManage) ||
          (access === 'editable' && document.canEdit);
        return matchesQuery && matchesType && matchesAccess;
      })
      .sort((left, right) => {
        if (sortOrder === 'name') {
          return left.title.localeCompare(right.title, undefined, { sensitivity: 'base' });
        }
        if (sortOrder === 'size') {
          return right.sizeBytes - left.sizeBytes;
        }
        return new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime();
      });
  }, [access, documents.items, fileType, query, sortOrder]);

  const hasActiveFilters = query !== '' || fileType !== 'all' || access !== 'all' || sortOrder !== 'updated';

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
          <label
            className={`upload-button ${isDragging ? 'upload-button-dragging' : ''}`}
            onDragEnter={() => setIsDragging(true)}
            onDragLeave={() => setIsDragging(false)}
            onDragOver={(event) => event.preventDefault()}
            onDrop={(event) => void onDrop(event)}
          >
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
        {documents.status === 'success' && documents.items.length > 0 ? (
          <section className="document-library-toolbar" aria-label="文档筛选与排序">
            <label className="library-search">
              <span>搜索文档</span>
              <input
                onChange={(event) => setQuery(event.target.value)}
                placeholder="按名称或原文件名搜索"
                type="search"
                value={query}
              />
            </label>
            <label>
              <span>文件类型</span>
              <select onChange={(event) => setFileType(event.target.value as FileTypeFilter)} value={fileType}>
                <option value="all">全部类型</option>
                <option value="docx">Word (.docx)</option>
                <option value="xlsx">Excel (.xlsx)</option>
                <option value="md">Markdown (.md)</option>
              </select>
            </label>
            <label>
              <span>我的权限</span>
              <select onChange={(event) => setAccess(event.target.value as AccessFilter)} value={access}>
                <option value="all">全部文档</option>
                <option value="editable">可编辑</option>
                <option value="managed">可管理</option>
              </select>
            </label>
            <label>
              <span>排序方式</span>
              <select onChange={(event) => setSortOrder(event.target.value as SortOrder)} value={sortOrder}>
                <option value="updated">最近更新</option>
                <option value="name">名称 A–Z</option>
                <option value="size">文件大小</option>
              </select>
            </label>
            {hasActiveFilters ? (
              <button
                className="secondary-button library-reset"
                onClick={() => {
                  setQuery('');
                  setFileType('all');
                  setAccess('all');
                  setSortOrder('updated');
                }}
                type="button"
              >
                清除筛选
              </button>
            ) : null}
          </section>
        ) : null}
        {documents.status === 'loading' ? <section className="empty-state">正在整理你的文档…</section> : null}
        {documents.status === 'error' ? (
          <section className="empty-state">{documents.error}</section>
        ) : null}
        {documents.status === 'success' && documents.items.length === 0 ? (
          <section className="empty-state">
            还没有文档。使用右上角的“上传”，或将 .docx、.xlsx、.md 文件拖放到上传按钮开始。
          </section>
        ) : null}
        {documents.status === 'success' && documents.items.length > 0 && filteredDocuments.length === 0 ? (
          <section className="empty-state" role="status">
            没有匹配的文档。请调整搜索词或清除筛选条件。
          </section>
        ) : null}
        {documents.status === 'success' && filteredDocuments.length > 0 ? (
          <section className="document-list">
            {filteredDocuments.map((document) => (
              <article className="document-row" key={document.id}>
                <div className="document-primary-meta">
                  <a className="document-title" href={`/documents/${document.id}`}>
                    {document.title}
                  </a>
                  <span>{document.canManage ? '可管理' : document.canEdit ? '可编辑' : '只读'}</span>
                </div>
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
