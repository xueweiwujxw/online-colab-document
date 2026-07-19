import { AuthProvider } from '../auth/AuthContext';
import { DocumentDetailPage } from '../pages/DocumentDetailPage/DocumentDetailPage';
import { DocumentListPage } from '../pages/DocumentListPage/DocumentListPage';
import { LoginPage } from '../pages/LoginPage/LoginPage';
import { MarkdownEditorPage } from '../pages/MarkdownEditorPage/MarkdownEditorPage';
import { OnlyOfficeEditorPage } from '../pages/OnlyOfficeEditorPage/OnlyOfficeEditorPage';
import { PermissionPage } from '../pages/PermissionPage/PermissionPage';
import { ShareAccessPage } from '../pages/ShareAccessPage/ShareAccessPage';
import { ShareManagementPage } from '../pages/ShareManagementPage/ShareManagementPage';
import './App.css';

export function App() {
  const path = window.location.pathname;
  const editorMatch = path.match(/^\/documents\/([^/]+)\/edit$/);
  const markdownMatch = path.match(/^\/documents\/([^/]+)\/markdown$/);
  const permissionMatch = path.match(/^\/documents\/([^/]+)\/permissions$/);
  const shareManagementMatch = path.match(/^\/documents\/([^/]+)\/share$/);
  const shareAccessMatch = path.match(/^\/share\/([^/]+)$/);
  const documentDetailMatch = path.match(/^\/documents\/([^/]+)$/);
  return (
    <AuthProvider>
      {shareAccessMatch ? <ShareAccessPage token={shareAccessMatch[1]} /> : null}
      {path === '/login' ? <LoginPage /> : null}
      {path === '/documents' || path === '/' ? <DocumentListPage /> : null}
      {editorMatch ? <OnlyOfficeEditorPage documentId={editorMatch[1]} /> : null}
      {markdownMatch ? <MarkdownEditorPage documentId={markdownMatch[1]} /> : null}
      {permissionMatch ? <PermissionPage documentId={permissionMatch[1]} /> : null}
      {shareManagementMatch ? <ShareManagementPage documentId={shareManagementMatch[1]} /> : null}
      {documentDetailMatch &&
      !permissionMatch &&
      !editorMatch &&
      !markdownMatch &&
      !shareManagementMatch ? (
        <DocumentDetailPage id={documentDetailMatch[1]} />
      ) : null}
      {path !== '/login' &&
      path !== '/documents' &&
      path !== '/' &&
      !shareAccessMatch &&
      !documentDetailMatch &&
      !permissionMatch &&
      !shareManagementMatch &&
      !markdownMatch &&
      !editorMatch ? (
        <DocumentListPage />
      ) : null}
    </AuthProvider>
  );
}
