import { FormEvent, useState } from 'react';

import { useAuth } from '../../auth/AuthContext';

export function LoginPage() {
  const auth = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await auth.login({ email, password });
      window.location.replace('/documents');
    } catch {
      setError('Email or password is incorrect.');
    } finally {
      setSubmitting(false);
    }
  }

  if (auth.status === 'authenticated') {
    window.location.replace('/documents');
    return null;
  }

  return (
    <main className="auth-shell">
      <form className="login-panel" onSubmit={onSubmit}>
        <div>
          <p className="eyebrow">Docs Collab Service</p>
          <h1>Sign in</h1>
        </div>

        <label className="field">
          <span>Email</span>
          <input
            autoComplete="email"
            disabled={submitting}
            inputMode="email"
            onChange={(event) => setEmail(event.target.value)}
            required
            type="email"
            value={email}
          />
        </label>

        <label className="field">
          <span>Password</span>
          <input
            autoComplete="current-password"
            disabled={submitting}
            minLength={8}
            onChange={(event) => setPassword(event.target.value)}
            required
            type="password"
            value={password}
          />
        </label>

        {error ? <p className="form-error">{error}</p> : null}

        <button className="primary-button" disabled={submitting} type="submit">
          {submitting ? 'Signing in' : 'Sign in'}
        </button>
      </form>
    </main>
  );
}
