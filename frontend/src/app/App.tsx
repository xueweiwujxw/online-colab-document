import { AuthProvider } from '../auth/AuthContext';
import { DocumentListPage } from '../pages/DocumentListPage/DocumentListPage';
import { LoginPage } from '../pages/LoginPage/LoginPage';
import './App.css';

export function App() {
  const path = window.location.pathname;
  return (
    <AuthProvider>
      {path === '/login' ? <LoginPage /> : <DocumentListPage />}
    </AuthProvider>
  );
}
