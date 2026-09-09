export function adminOfficerDashboardPath(employee) {
  const id = Number(employee?.id)
  if (employee?.role !== 'officer' || !Number.isInteger(id) || id <= 0) return null
  return `/admin/officers/${id}`
}

export function openAdminOfficerDashboard(employee, navigate) {
  const path = adminOfficerDashboardPath(employee)
  if (path) navigate(path)
  return path
}

export function adminOfficerDashboardEndpoints(officerId) {
  const id = Number(officerId)
  if (!Number.isInteger(id) || id <= 0 || String(id) !== String(officerId)) return null
  const base = `/admin/officers/${id}`
  return {
    dashboard: `${base}/dashboard`,
    targetOverview: `${base}/target-overview`,
  }
}
