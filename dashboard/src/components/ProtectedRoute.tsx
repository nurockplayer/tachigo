import { Navigate, Outlet } from 'react-router'

export default function ProtectedRoute() {
  const token = localStorage.getItem('token')

  if (!token) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}
