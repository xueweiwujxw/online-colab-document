import { FormEvent, useState } from 'react';

import { ApiError } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';

export function RegisterPage() {
  const auth = useAuth();
  const [displayName, setDisplayName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await auth.register({ displayName, email, password });
      window.location.replace('/documents');
    } catch (registerError) {
      if (registerError instanceof ApiError && registerError.status === 409) {
        setError('这个邮箱已经注册。');
      } else if (registerError instanceof ApiError && registerError.status === 400) {
        setError('请填写有效邮箱、显示名和至少 8 位密码。');
      } else {
        setError('注册失败，请稍后重试。');
      }
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
          <h1>注册账户</h1>
        </div>

        <label className="field">
          <span>显示名</span>
          <input
            autoComplete="name"
            disabled={submitting}
            onChange={(event) => setDisplayName(event.target.value)}
            required
            type="text"
            value={displayName}
          />
        </label>

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
            autoComplete="new-password"
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
          {submitting ? '注册中' : '注册并登录'}
        </button>

        <p className="auth-switch">
          已有账户？ <a href="/login">去登录</a>
        </p>
      </form>
    </main>
  );
}
