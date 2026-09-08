export function defaultEntryStatusFilter(canManageEntries) {
  return canManageEntries ? 'pending' : ''
}

export function entryMatchesStatus(entry, statusFilter) {
  return !statusFilter || entry.status === statusFilter
}

export function reconcileEntryStatus(entries, updatedEntry, statusFilter) {
  const existingIndex = entries.findIndex((entry) => entry.id === updatedEntry.id)

  if (!entryMatchesStatus(updatedEntry, statusFilter)) {
    return existingIndex === -1
      ? entries
      : entries.filter((entry) => entry.id !== updatedEntry.id)
  }

  if (existingIndex === -1) return [updatedEntry, ...entries]
  return entries.map((entry) => (entry.id === updatedEntry.id ? updatedEntry : entry))
}
