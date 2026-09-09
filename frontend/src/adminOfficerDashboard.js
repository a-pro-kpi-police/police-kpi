export function adminOfficerDashboardPath(employee) {
  if (employee?.role !== 'officer') return null
  return adminOfficerDashboardPathByID(employee.id)
}

export function adminOfficerDashboardPathByID(officerId) {
  const id = Number(officerId)
  if (!Number.isInteger(id) || id <= 0) return null
  return `/admin/officers/${id}`
}

export function openAdminOfficerDashboard(employee, navigate) {
  const path = adminOfficerDashboardPath(employee)
  if (path) navigate(path)
  return path
}

export function openAdminRatingOfficerDashboard(officer, navigate) {
  const path = adminOfficerDashboardPathByID(officer?.officer_id)
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
