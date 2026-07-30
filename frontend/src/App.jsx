import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './auth.jsx'
import Layout from './components/Layout.jsx'
import Login from './pages/Login.jsx'
import Messenger from './pages/Messenger.jsx'
import Dashboard from './pages/Dashboard.jsx'
import Entries from './pages/Entries.jsx'
import Officers from './pages/Officers.jsx'
import Targets from './pages/Targets.jsx'

function Protected({ children }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="loading">Загрузка…</div>
  if (!user) return <Navigate to="/login" replace />
  return children
}

function HeadOnly({ children }) {
  const { user } = useAuth()
  if (user?.role !== 'head') return <Navigate to="/entries" replace />
  return children
}

function HomeRedirect() {
  const { user } = useAuth()
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
        <Route path="messenger" element={<Messenger />} />
        <Route path="targets" element={<Targets />} />
        <Route
          path="officers"
          element={
            <HeadOnly>
              <Officers />
            </HeadOnly>
          }
        />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
