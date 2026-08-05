import { ChangeEvent, FormEvent, useEffect, useState } from 'react';

import {
  changePassword,
  listSessions,
  revokeSession,
  updateProfile,
  updateAvatar,
  type AuthSession,
} from '../../api/auth';
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
  const [sessionState, setSessionState] = useState<{
    status: 'idle' | 'loading' | 'success' | 'error';
    items: AuthSession[];
    error: string | null;
  }>({ status: 'idle', items: [], error: null });
  const [revokingSessionId, setRevokingSessionId] = useState<string | null>(null);
  const [avatarSubmitting, setAvatarSubmitting] = useState(false);
  const [avatarError, setAvatarError] = useState<string | null>(null);

  useEffect(() => {
    if (auth.status === 'authenticated') {
      setDisplayName(auth.user.displayName);
      void refreshSessions();
    }
  }, [auth.status, auth.status === 'authenticated' ? auth.user.displayName : null]);

  async function refreshSessions() {
    setSessionState((current) => ({ status: 'loading', items: current.items, error: null }));
    try {
      const items = await listSessions();
      setSessionState({ status: 'success', items, error: null });
    } catch (caught) {
      setSessionState({ status: 'error', items: [], error: errorMessage(caught, '加载会话失败') });
    }
  }

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

  async function onAvatarChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type) || file.size > 2 * 1024 * 1024) {
      setAvatarError('请选择不超过 2MB 的 PNG、JPEG 或 WebP 图片。');
      return;
    }
    setAvatarSubmitting(true);
    setAvatarError(null);
    try {
      await updateAvatar(file);
      await auth.refresh();
    } catch (caught) {
      setAvatarError(errorMessage(caught, '上传头像失败'));
    } finally {
      setAvatarSubmitting(false);
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

  async function onRevokeSession(session: AuthSession) {
    if (session.current) {
      return;
    }
    if (!window.confirm('确认撤销这个登录会话？')) {
      return;
    }
    setRevokingSessionId(session.id);
    try {
      await revokeSession(session.id);
      await refreshSessions();
    } catch (caught) {
      setSessionState({ status: 'error', items: sessionState.items, error: errorMessage(caught, '撤销会话失败') });
    } finally {
      setRevokingSessionId(null);
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
          <div className="profile-avatar-row">
            {auth.user.avatarUrl ? <img alt={`${auth.user.displayName} 的头像`} className="profile-avatar" src={auth.user.avatarUrl} /> : <span aria-hidden="true" className="profile-avatar profile-avatar-fallback">{auth.user.displayName.slice(0, 1).toUpperCase()}</span>}
            <label className="secondary-button avatar-upload-button">
              <input accept="image/jpeg,image/png,image/webp" disabled={avatarSubmitting} onChange={onAvatarChange} type="file" />
              {avatarSubmitting ? '上传中' : '更换头像'}
            </label>
          </div>
          {avatarError ? <p className="form-error">{avatarError}</p> : null}
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

        <article className="profile-panel">
          <p className="eyebrow">登录</p>
          <h2>会话管理</h2>
          {sessionState.status === 'loading' ? <p className="empty-inline">加载中</p> : null}
          {sessionState.status === 'error' ? <p className="form-error">{sessionState.error}</p> : null}
          {sessionState.status === 'success' && sessionState.items.length === 0 ? (
            <p className="empty-inline">暂无登录会话。</p>
          ) : null}
          {sessionState.items.length > 0 ? (
            <div className="session-list">
              {sessionState.items.map((session) => (
                <div className="session-row" key={session.id}>
                  <div>
                    <strong>{session.current ? '当前会话' : '其他会话'}</strong>
                    <span className="document-meta">创建于 {formatDate(session.createdAt)}</span>
                    <span className="document-meta">过期于 {formatDate(session.expiresAt)}</span>
                  </div>
                  <button
                    className="secondary-button danger-button"
                    disabled={session.current || revokingSessionId === session.id}
                    onClick={() => void onRevokeSession(session)}
                    type="button"
                  >
                    {revokingSessionId === session.id ? '撤销中' : '撤销'}
                  </button>
                </div>
              ))}
            </div>
          ) : null}
        </article>
      </section>
    </main>
  );
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
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
