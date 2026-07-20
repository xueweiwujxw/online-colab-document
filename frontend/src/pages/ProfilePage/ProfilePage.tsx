import { FormEvent, useState } from 'react';

import { changePassword } from '../../api/auth';
import { ApiError, errorMessage } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';

export function ProfilePage() {
  const auth = useAuth();
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setSuccess(null);
    if (newPassword.length < 8) {
      setError('新密码至少需要 8 位。');
      return;
    }
    if (newPassword !== confirmPassword) {
      setError('两次输入的新密码不一致。');
      return;
    }
    setSubmitting(true);
    try {
      await changePassword({ currentPassword, newPassword });
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
      setSuccess('密码已更新。');
    } catch (caught) {
      if (caught instanceof ApiError && caught.status === 401) {
        setError('当前密码不正确。');
      } else if (caught instanceof ApiError && caught.status === 400) {
        setError('当前账户不支持在这里修改密码。');
      } else {
        setError(errorMessage(caught, '修改密码失败'));
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (auth.status === 'loading') {
    return (
      <main className="app-shell">
        <section className="empty-state">加载中</section>
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
          <p className="eyebrow">账户</p>
          <h1>用户中心</h1>
        </div>
        <div className="user-actions">
          <a className="secondary-button" href="/documents">
            文档
          </a>
          <button className="secondary-button" onClick={() => void auth.logout()} type="button">
            退出登录
          </button>
        </div>
      </header>

      <section className="profile-layout">
        <article className="profile-panel">
          <p className="eyebrow">基本信息</p>
          <dl className="profile-details">
            <div>
              <dt>显示名</dt>
              <dd>{auth.user.displayName}</dd>
            </div>
            <div>
              <dt>邮箱</dt>
              <dd>{auth.user.email}</dd>
            </div>
            <div>
              <dt>登录方式</dt>
              <dd>{authSourceLabel(auth.user.authSource)}</dd>
            </div>
            <div>
              <dt>角色</dt>
              <dd>{auth.user.isAdmin ? '管理员' : '普通用户'}</dd>
            </div>
          </dl>
        </article>

        <article className="profile-panel">
          <p className="eyebrow">安全</p>
          <h2>修改密码</h2>
          {auth.user.authSource !== 'local' ? (
            <p className="empty-inline">OIDC 登录账户的密码由身份提供方管理。</p>
          ) : (
            <form className="profile-form" onSubmit={onSubmit}>
              <label className="field">
                <span>当前密码</span>
                <input
                  autoComplete="current-password"
                  disabled={submitting}
                  onChange={(event) => setCurrentPassword(event.target.value)}
                  required
                  type="password"
                  value={currentPassword}
                />
              </label>
              <label className="field">
                <span>新密码</span>
                <input
                  autoComplete="new-password"
                  disabled={submitting}
                  minLength={8}
                  onChange={(event) => setNewPassword(event.target.value)}
                  required
                  type="password"
                  value={newPassword}
                />
              </label>
              <label className="field">
                <span>确认新密码</span>
                <input
                  autoComplete="new-password"
                  disabled={submitting}
                  minLength={8}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  required
                  type="password"
                  value={confirmPassword}
                />
              </label>
              {error ? <p className="form-error">{error}</p> : null}
              {success ? <p className="form-success">{success}</p> : null}
              <button className="primary-button profile-submit" disabled={submitting} type="submit">
                {submitting ? '保存中' : '保存新密码'}
              </button>
            </form>
          )}
        </article>
      </section>
    </main>
  );
}

function authSourceLabel(authSource: string): string {
  if (authSource === 'local') {
    return '本地账号';
  }
  if (authSource === 'oidc') {
    return 'OIDC';
  }
  return authSource;
}
