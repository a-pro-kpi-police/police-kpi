import { useI18n } from '../i18n.jsx'

export default function StatusBadge({ status }) {
  const { t } = useI18n()
  return <span className={`badge badge-${status}`}>{t(status)}</span>
}
