import { useAuth } from '../../auth/AuthContext';

export function DocumentListPage() {
  const auth = useAuth();

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
          <span>{auth.user.displayName}</span>
          <button className="secondary-button" onClick={() => void auth.logout()} type="button">
            Sign out
          </button>
        </div>
      </header>
      <section className="empty-state">No documents yet.</section>
    </main>
  );
}
