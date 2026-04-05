export type UserRole = 'viewer' | 'streamer' | 'agency' | 'admin'

export const ROLE_LABELS: Record<UserRole, string> = {
  viewer: '觀眾',
  streamer: '實況主',
  agency: '經紀公司',
  admin: '管理員',
}

export const ROLE_BADGE_STYLES: Record<UserRole, string> = {
  viewer: 'bg-slate-500/10 text-slate-700',
  streamer: 'bg-sky-500/10 text-sky-700',
  agency: 'bg-amber-500/10 text-amber-700',
  admin: 'bg-emerald-500/10 text-emerald-700',
}
