import { useTranslation } from 'react-i18next'

interface PointsDisplayProps {
  spendableBalance: number | null
  cumulativeTotal: number | null
  gain: number | null
  isAnimating: boolean
}

function formatBalance(value: number | null) {
  return value?.toLocaleString() ?? '...'
}

export function PointsDisplay({
  spendableBalance,
  cumulativeTotal,
  gain,
  isAnimating,
}: PointsDisplayProps) {
  const { t } = useTranslation()

  return (
    <section className="ext-balance-wrap" aria-label={t('hud.pointsBalance')}>
      <div className={`ext-balance ${isAnimating ? 'ext-balance--bump' : ''}`}>
        <div className="ext-balance__metric">
          <span className="ext-balance__label">{t('hud.availableBalance')}</span>
          <strong className="ext-balance__value">{formatBalance(spendableBalance)}</strong>
        </div>
        <div className="ext-balance__metric ext-balance__metric--secondary">
          <span className="ext-balance__label">{t('hud.cumulativeTotal')}</span>
          <strong className="ext-balance__value">{formatBalance(cumulativeTotal)}</strong>
        </div>
      </div>
      {gain !== null && gain > 0 && (
        <span className="ext-balance-gain">+{gain.toLocaleString()} {t('common.points')}</span>
      )}
    </section>
  )
}
