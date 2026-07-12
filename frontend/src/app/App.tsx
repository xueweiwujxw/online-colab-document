import { AuthProvider } from '../auth/AuthContext';
import { DocumentDetailPage } from '../pages/DocumentDetailPage/DocumentDetailPage';
import { DocumentListPage } from '../pages/DocumentListPage/DocumentListPage';
import { LoginPage } from '../pages/LoginPage/LoginPage';
import './App.css';

export function App() {
  const path = window.location.pathname;
  const documentDetailMatch = path.match(/^\/documents\/([^/]+)$/);
  return (
    <AuthProvider>
      {path === '/login' ? <LoginPage /> : null}
      {path === '/documents' || path === '/' ? <DocumentListPage /> : null}
      {documentDetailMatch ? <DocumentDetailPage id={documentDetailMatch[1]} /> : null}
      {path !== '/login' && path !== '/documents' && path !== '/' && !documentDetailMatch ? (
        <DocumentListPage />
      ) : null}
    </AuthProvider>
  );
}
