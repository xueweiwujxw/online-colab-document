import { type ReactNode } from 'react';

import { useAuth } from '../../auth/AuthContext';

export type AdminSection = 'overview' | 'users' | 'documents' | 'storage' | 'identity' | 'audit';

const navigation: Array<{ id: AdminSection; href: string; label: string; note: string }> = [
  { id: 'overview', href: '/admin', label: '概览', note: '系统状态' },
  { id: 'users', href: '/admin/users', label: '用户与访问', note: '账号和角色' },
  { id: 'documents', href: '/admin/documents', label: '文件治理', note: '全局文档' },
  { id: 'storage', href: '/admin/storage', label: '对象存储', note: 'MinIO 浏览器' },
  { id: 'identity', href: '/admin/identity', label: '身份连接', note: 'OIDC 状态' },
  { id: 'audit', href: '/admin/audit-logs', label: '审计日志', note: '事件检索' },
];

export function AdminShell({ section, title, description, children }: { section: AdminSection; title: string; description: string; children: ReactNode }) {
  const auth = useAuth();
  if (auth.status === 'loading') return <main className="app-shell"><section className="empty-state">加载中</section></main>;
  if (auth.status === 'anonymous') { window.location.replace('/login'); return null; }
  if (!auth.user.isAdmin) return <main className="app-shell"><section className="empty-state">无权限访问</section></main>;
  return <main className="admin-shell"><aside className="admin-sidebar"><a className="admin-brand" href="/admin"><span>DC</span><strong>控制台</strong></a><nav aria-label="管理导航" className="admin-nav">{navigation.map((item) => <a aria-current={section === item.id ? 'page' : undefined} className={section === item.id ? 'admin-nav-item admin-nav-item-current' : 'admin-nav-item'} href={item.href} key={item.id}><strong>{item.label}</strong><small>{item.note}</small></a>)}</nav><div className="admin-sidebar-footer"><span>{auth.user.displayName}</span><a href="/documents">返回工作区</a></div></aside><section className="admin-workspace"><header className="admin-workspace-header"><div><p className="eyebrow">管理后台</p><h1>{title}</h1><p>{description}</p></div><div className="user-actions"><a className="secondary-button" href="/profile">账户</a><button className="secondary-button" onClick={() => void auth.logout()} type="button">退出</button></div></header>{children}</section></main>;
}
