import { Authenticated, Refine } from '@refinedev/core'
import routerProvider from '@refinedev/react-router'
import { createBrowserRouter, Navigate, Outlet, RouterProvider, useParams } from 'react-router'
import Layout from '@/components/Layout'
import LoginPage from '@/pages/LoginPage'
import DashboardPage from '@/pages/DashboardPage'
import StreamersPage from '@/pages/StreamersPage'
import StreamerDetailPage from '@/pages/StreamerDetailPage'
import TransactionsPage from '@/pages/TransactionsPage'
import SettingsPage from '@/pages/SettingsPage'
import RafflesPage from '@/pages/RafflesPage'
import RaffleDetailPage from '@/pages/RaffleDetailPage'
import { authProvider } from '@/providers/authProvider'
import { dataProvider } from '@/providers/dataProvider'

const router = createBrowserRouter([
  {
    Component: RefineRoot,
    children: [
      {
        path: '/login',
        element: <LoginPage />,
      },
      {
        element: <Layout />,
        children: [
          {
            element: (
              <Authenticated key="authenticated-routes" redirectOnFail="/login">
                <Outlet />
              </Authenticated>
            ),
            children: [
              { index: true, element: <DashboardPage /> },
              { path: 'streamers', element: <StreamersPage /> },
              { path: 'streamers/:streamerId', element: <StreamerDetailRoute /> },
              { path: 'raffles', element: <RafflesPage /> },
              { path: 'raffles/:raffleId', element: <RaffleDetailPage /> },
              { path: 'transactions', element: <TransactionsPage /> },
              { path: 'settings', element: <SettingsPage /> },
            ],
          },
        ],
      },
      {
        path: '*',
        element: <Navigate to="/" replace />,
      },
    ],
  },
])

function StreamerDetailRoute() {
  const { streamerId } = useParams()
  return <StreamerDetailPage key={streamerId} />
}

function RefineRoot() {
  return (
    <Refine
      authProvider={authProvider}
      dataProvider={dataProvider}
      routerProvider={routerProvider}
      resources={[
        { name: 'streamers', list: '/streamers', show: '/streamers/:streamerId' },
        { name: 'raffles', list: '/raffles', show: '/raffles/:raffleId' },
        { name: 'transactions', list: '/transactions' },
        { name: 'settings', list: '/settings' },
      ]}
    >
      <Outlet />
    </Refine>
  )
}

export default function App() {
  return <RouterProvider router={router} />
}
