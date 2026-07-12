import { AuthProvider } from '../auth/AuthContext';
import { DocumentDetailPage } from '../pages/DocumentDetailPage/DocumentDetailPage';
import { DocumentListPage } from '../pages/DocumentListPage/DocumentListPage';
import { LoginPage } from '../pages/LoginPage/LoginPage';
import { OnlyOfficeEditorPage } from '../pages/OnlyOfficeEditorPage/OnlyOfficeEditorPage';
import { PermissionPage } from '../pages/PermissionPage/PermissionPage';
import './App.css';

export function App() {
  const path = window.location.pathname;
  const editorMatch = path.match(/^\/documents\/([^/]+)\/edit$/);
  const permissionMatch = path.match(/^\/documents\/([^/]+)\/permissions$/);
  const documentDetailMatch = path.match(/^\/documents\/([^/]+)$/);
  return (
    <AuthProvider>
      {path === '/login' ? <LoginPage /> : null}
      {path === '/documents' || path === '/' ? <DocumentListPage /> : null}
      {editorMatch ? <OnlyOfficeEditorPage documentId={editorMatch[1]} /> : null}
      {permissionMatch ? <PermissionPage documentId={permissionMatch[1]} /> : null}
      {documentDetailMatch && !permissionMatch && !editorMatch ? (
        <DocumentDetailPage id={documentDetailMatch[1]} />
      ) : null}
      {path !== '/login' &&
      path !== '/documents' &&
      path !== '/' &&
      !documentDetailMatch &&
      !permissionMatch &&
      !editorMatch ? (
        <DocumentListPage />
      ) : null}
    </AuthProvider>
  );
}
