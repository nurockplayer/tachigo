import { NavLink, Outlet } from 'react-router'
import { LayoutDashboard, Users, ArrowLeftRight, Settings } from 'lucide-react'
import { cn } from '@/lib/utils'

const navItems = [
  { to: '/', label: '總覽', icon: LayoutDashboard, end: true },
  { to: '/streamers', label: '實況主管理', icon: Users, end: false },
  { to: '/transactions', label: '交易紀錄', icon: ArrowLeftRight, end: false },
  { to: '/settings', label: '設定', icon: Settings, end: false },
]

export default function Layout() {
  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar */}
      <aside className="w-60 shrink-0 border-r border-border bg-secondary/30 flex flex-col">
        <div className="h-16 flex items-center px-6 border-b border-border">
          <span className="text-lg font-bold text-foreground">Tachigo</span>
        </div>
        <nav className="flex-1 px-3 py-4 space-y-1">
          {navItems.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                )
              }
            >
              <Icon size={16} />
              {label}
            </NavLink>
          ))}
        </nav>
      </aside>

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        <header className="h-16 shrink-0 border-b border-border flex items-center px-6">
          <span className="text-sm text-muted-foreground">Tachigo Dashboard</span>
        </header>
        <main className="flex-1 overflow-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
