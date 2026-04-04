import { Authenticated, Refine } from '@refinedev/core'
import routerProvider from '@refinedev/react-router'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router'
import Layout from '@/components/Layout'
import LoginPage from '@/pages/LoginPage'
import DashboardPage from '@/pages/DashboardPage'
import StreamerDetailPage from '@/pages/StreamerDetailPage'
import StreamersPage from '@/pages/StreamersPage'
import TransactionsPage from '@/pages/TransactionsPage'
import SettingsPage from '@/pages/SettingsPage'
import { authProvider } from '@/services/authProvider'
import { dataProvider } from '@/services/dataProvider'

export default function App() {
  return (
    <BrowserRouter>
      <Refine
        authProvider={authProvider}
        dataProvider={dataProvider}
        routerProvider={routerProvider}
      >
        <Routes>
          <Route
            path="/login"
            element={
              <Authenticated key="login" fallback={<LoginPage />}>
                <Navigate to="/" replace />
              </Authenticated>
            }
          />
          <Route
            element={
              <Authenticated key="protected" fallback={<Navigate to="/login" replace />}>
                <Layout />
              </Authenticated>
            }
          >
            <Route index element={<DashboardPage />} />
            <Route path="/streamers" element={<StreamersPage />} />
            <Route path="/streamers/:streamerId" element={<StreamerDetailPage />} />
            <Route path="/transactions" element={<TransactionsPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Refine>
    </BrowserRouter>
  )
}
