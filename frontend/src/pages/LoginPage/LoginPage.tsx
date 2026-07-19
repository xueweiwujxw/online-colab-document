import { FormEvent, useState } from 'react';

import { getOIDCLoginURL } from '../../api/auth';
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
      setError('邮箱或密码不正确。');
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
          <p className="eyebrow">在线协作文档</p>
          <h1>登录</h1>
        </div>

        <label className="field">
          <span>邮箱</span>
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
          <span>密码</span>
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
          {submitting ? '登录中' : '登录'}
        </button>

        <div className="divider">
          <span>或</span>
        </div>

        <button
          className="secondary-button oidc-button"
          disabled={submitting}
          onClick={() => window.location.assign(getOIDCLoginURL())}
          type="button"
        >
          使用 OIDC 登录
        </button>
      </form>
    </main>
  );
}
