import { AuthProvider } from '../auth/AuthContext';
import { AuditLogPage } from '../pages/AuditLogPage/AuditLogPage';
import { AdminPage } from '../pages/AdminPage/AdminPage';
import { DocumentDetailPage } from '../pages/DocumentDetailPage/DocumentDetailPage';
import { DocumentListPage } from '../pages/DocumentListPage/DocumentListPage';
import { LoginPage } from '../pages/LoginPage/LoginPage';
import { MarkdownEditorPage } from '../pages/MarkdownEditorPage/MarkdownEditorPage';
import { OfficeEditorPage } from '../pages/OfficeEditorPage/OfficeEditorPage';
import { PermissionPage } from '../pages/PermissionPage/PermissionPage';
import { ProfilePage } from '../pages/ProfilePage/ProfilePage';
import { RegisterPage } from '../pages/RegisterPage/RegisterPage';
import { ShareAccessPage } from '../pages/ShareAccessPage/ShareAccessPage';
import { ShareManagementPage } from '../pages/ShareManagementPage/ShareManagementPage';
import { VersionPage } from '../pages/VersionPage/VersionPage';
import './App.css';

export function App() {
  const path = window.location.pathname;
  const editorMatch = path.match(/^\/documents\/([^/]+)\/edit$/);
  const markdownMatch = path.match(/^\/documents\/([^/]+)\/markdown$/);
  const permissionMatch = path.match(/^\/documents\/([^/]+)\/permissions$/);
  const shareManagementMatch = path.match(/^\/documents\/([^/]+)\/share$/);
  const versionMatch = path.match(/^\/documents\/([^/]+)\/versions$/);
  const shareAccessMatch = path.match(/^\/share\/([^/]+)$/);
  const documentDetailMatch = path.match(/^\/documents\/([^/]+)$/);
  return (
    <AuthProvider>
      {shareAccessMatch ? <ShareAccessPage token={shareAccessMatch[1]} /> : null}
      {path === '/admin/audit-logs' ? <AuditLogPage /> : null}
      {path === '/admin' ? <AdminPage /> : null}
      {path === '/login' ? <LoginPage /> : null}
      {path === '/register' ? <RegisterPage /> : null}
      {path === '/profile' ? <ProfilePage /> : null}
      {path === '/documents' || path === '/' ? <DocumentListPage /> : null}
      {editorMatch ? <OfficeEditorPage documentId={editorMatch[1]} /> : null}
      {markdownMatch ? <MarkdownEditorPage documentId={markdownMatch[1]} /> : null}
      {permissionMatch ? <PermissionPage documentId={permissionMatch[1]} /> : null}
      {shareManagementMatch ? <ShareManagementPage documentId={shareManagementMatch[1]} /> : null}
      {versionMatch ? <VersionPage documentId={versionMatch[1]} /> : null}
      {documentDetailMatch &&
      !permissionMatch &&
      !editorMatch &&
      !markdownMatch &&
      !shareManagementMatch &&
      !versionMatch ? (
        <DocumentDetailPage id={documentDetailMatch[1]} />
      ) : null}
      {path !== '/login' &&
      path !== '/register' &&
      path !== '/profile' &&
      path !== '/documents' &&
      path !== '/' &&
      path !== '/admin/audit-logs' &&
      path !== '/admin' &&
      !shareAccessMatch &&
      !documentDetailMatch &&
      !permissionMatch &&
      !shareManagementMatch &&
      !versionMatch &&
      !markdownMatch &&
      !editorMatch ? (
        <DocumentListPage />
      ) : null}
    </AuthProvider>
  );
}
