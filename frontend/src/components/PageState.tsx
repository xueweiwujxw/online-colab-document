import { type ReactNode } from 'react';

import { useAuth } from '../auth/AuthContext';

export function PageLoading() {
  return (
    <main className="app-shell">
      <section className="empty-state">Loading</section>
    </main>
  );
}

export function PageError({ children }: { children: ReactNode }) {
  return (
    <main className="app-shell">
      <a className="back-link" href="/documents">
        Back to documents
      </a>
      <section className="empty-state">{children}</section>
    </main>
  );
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const auth = useAuth();
  if (auth.status === 'loading') {
    return <PageLoading />;
  }
  if (auth.status === 'anonymous') {
    window.location.replace('/login');
    return null;
  }
  return <>{children}</>;
}
