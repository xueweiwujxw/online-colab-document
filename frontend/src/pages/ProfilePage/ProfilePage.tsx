import { FormEvent, useEffect, useState } from 'react';

import { changePassword, updateProfile } from '../../api/auth';
import { ApiError, errorMessage } from '../../api/client';
import { useAuth } from '../../auth/AuthContext';

export function ProfilePage() {
  const auth = useAuth();
  const [displayName, setDisplayName] = useState('');
  const [profileSubmitting, setProfileSubmitting] = useState(false);
  const [profileError, setProfileError] = useState<string | null>(null);
  const [profileSuccess, setProfileSuccess] = useState<string | null>(null);
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordSubmitting, setPasswordSubmitting] = useState(false);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState<string | null>(null);

  useEffect(() => {
    if (auth.status === 'authenticated') {
      setDisplayName(auth.user.displayName);
    }
  }, [auth.status, auth.status === 'authenticated' ? auth.user.displayName : null]);

  async function onProfileSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setProfileError(null);
    setProfileSuccess(null);
    const nextDisplayName = displayName.trim();
    if (!nextDisplayName) {
      setProfileError('显示名不能为空。');
      return;
    }
    setProfileSubmitting(true);
    try {
      await updateProfile({ displayName: nextDisplayName });
      await auth.refresh();
      setDisplayName(nextDisplayName);
      setProfileSuccess('基本信息已更新。');
    } catch (caught) {
      setProfileError(errorMessage(caught, '更新基本信息失败'));
    } finally {
      setProfileSubmitting(false);
    }
  }

  async function onPasswordSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPasswordError(null);
    setPasswordSuccess(null);
    if (newPassword.length < 8) {
      setPasswordError('新密码至少需要 8 位。');
      return;
    }
    if (newPassword !== confirmPassword) {
      setPasswordError('两次输入的新密码不一致。');
      return;
    }
    setPasswordSubmitting(true);
    try {
      await changePassword({ currentPassword, newPassword });
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
      setPasswordSuccess('密码已更新。');
    } catch (caught) {
      if (caught instanceof ApiError && caught.status === 401) {
        setPasswordError('当前密码不正确。');
      } else if (caught instanceof ApiError && caught.status === 400) {
        setPasswordError('当前账户不支持在这里修改密码。');
      } else {
        setPasswordError(errorMessage(caught, '修改密码失败'));
      }
    } finally {
      setPasswordSubmitting(false);
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
          <form className="profile-form profile-edit-form" onSubmit={onProfileSubmit}>
            <label className="field">
              <span>修改显示名</span>
              <input
                autoComplete="name"
                disabled={profileSubmitting}
                maxLength={120}
                onChange={(event) => setDisplayName(event.target.value)}
                required
                type="text"
                value={displayName}
              />
            </label>
            {profileError ? <p className="form-error">{profileError}</p> : null}
            {profileSuccess ? <p className="form-success">{profileSuccess}</p> : null}
            <button className="primary-button profile-submit" disabled={profileSubmitting} type="submit">
              {profileSubmitting ? '保存中' : '保存基本信息'}
            </button>
          </form>
        </article>

        <article className="profile-panel">
          <p className="eyebrow">安全</p>
          <h2>修改密码</h2>
          {auth.user.authSource !== 'local' ? (
            <p className="empty-inline">OIDC 登录账户的密码由身份提供方管理。</p>
          ) : (
            <form className="profile-form" onSubmit={onPasswordSubmit}>
              <label className="field">
                <span>当前密码</span>
                <input
                  autoComplete="current-password"
                  disabled={passwordSubmitting}
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
                  disabled={passwordSubmitting}
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
                  disabled={passwordSubmitting}
                  minLength={8}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  required
                  type="password"
                  value={confirmPassword}
                />
              </label>
              {passwordError ? <p className="form-error">{passwordError}</p> : null}
              {passwordSuccess ? <p className="form-success">{passwordSuccess}</p> : null}
              <button className="primary-button profile-submit" disabled={passwordSubmitting} type="submit">
                {passwordSubmitting ? '保存中' : '保存新密码'}
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
