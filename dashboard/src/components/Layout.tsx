import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router'
import { LayoutDashboard, Users, ArrowLeftRight, Settings } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { logout } from '@/services/auth'

const navItems = [
  { to: '/', label: '總覽', icon: LayoutDashboard, end: true },
  { to: '/streamers', label: '實況主管理', icon: Users, end: false },
  { to: '/transactions', label: '交易紀錄', icon: ArrowLeftRight, end: false },
  { to: '/settings', label: '設定', icon: Settings, end: false },
]

export default function Layout() {
  const navigate = useNavigate()
  const [isLoggingOut, setIsLoggingOut] = useState(false)

  async function handleLogout() {
    setIsLoggingOut(true)
    await logout()
    navigate('/login')
  }

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
        <header className="flex h-16 shrink-0 items-center justify-between border-b border-border px-6">
          <span className="text-sm text-muted-foreground">Tachigo Dashboard</span>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleLogout}
            disabled={isLoggingOut}
          >
            {isLoggingOut ? '登出中...' : '登出'}
          </Button>
        </header>
        <main className="flex-1 overflow-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
