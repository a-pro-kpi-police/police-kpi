export function rolePermissions(role) {
  const isAdmin = role === 'admin'
  const isHead = role === 'head'
  const isLeadership = role === 'leadership'
  return {
    isAdmin,
    isHead,
    isLeadership,
    canManageEntries: isHead || isAdmin,
    canViewEntries: isHead || isAdmin || isLeadership,
    canViewOverview: isHead || isLeadership,
    canUseCommunication: role === 'officer' || isHead,
  }
}
