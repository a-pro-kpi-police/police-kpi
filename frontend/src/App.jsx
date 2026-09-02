import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './auth.jsx'
import Layout from './components/Layout.jsx'
import Login from './pages/Login.jsx'
import Messenger from './pages/Messenger.jsx'
import Dashboard from './pages/Dashboard.jsx'
import Entries from './pages/Entries.jsx'
import Officers from './pages/Officers.jsx'
import Targets from './pages/Targets.jsx'
import Admin from './pages/Admin.jsx'

function Protected({ children }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="loading">Загрузка…</div>
  if (!user) return <Navigate to="/login" replace />
  return children
}

function HeadOnly({ children }) {
  const { user } = useAuth()
  if (user?.role !== 'head') return <Navigate to={user?.role === 'admin' ? '/admin' : '/entries'} replace />
  return children
}

function AdminOnly({ children }) {
  const { user } = useAuth()
  if (user?.role !== 'admin') return <Navigate to={user?.role === 'head' ? '/dashboard' : '/entries'} replace />
  return children
}

function MessengerOnly({ children }) {
  const { user } = useAuth()
  if (user?.role === 'admin') return <Navigate to="/admin" replace />
  return children
}

function HomeRedirect() {
  const { user } = useAuth()
  if (user?.role === 'admin') return <Navigate to="/admin" replace />
  return <Navigate to={user?.role === 'head' ? '/dashboard' : '/entries'} replace />
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <Protected>
            <Layout />
          </Protected>
        }
      >
        <Route index element={<HomeRedirect />} />
        <Route
          path="dashboard"
          element={
            <HeadOnly>
              <Dashboard />
            </HeadOnly>
          }
        />
        <Route
          path="dashboard/officers/:officerId"
          element={
            <HeadOnly>
              <Dashboard />
            </HeadOnly>
          }
        />
        <Route path="entries" element={<Entries />} />
        <Route
          path="messenger"
          element={
            <MessengerOnly>
              <Messenger />
            </MessengerOnly>
          }
        />
        <Route
          path="targets"
          element={
            <HeadOnly>
              <Targets />
            </HeadOnly>
          }
        />
        <Route
          path="officers"
          element={
            <HeadOnly>
              <Officers />
            </HeadOnly>
          }
        />
        <Route
          path="admin"
          element={
            <AdminOnly>
              <Admin />
            </AdminOnly>
          }
        />
        <Route
          path="admin/catalog"
          element={
            <AdminOnly>
              <Admin initialTab="catalog" />
            </AdminOnly>
          }
        />
        <Route
          path="admin/rating"
          element={
            <AdminOnly>
              <Admin initialTab="rating" />
            </AdminOnly>
          }
        />
        <Route
          path="admin/support"
          element={
            <AdminOnly>
              <Admin initialTab="support" />
            </AdminOnly>
          }
        />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
