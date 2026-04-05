import { useTwitch } from './hooks/useTwitch'
import { useBits } from './hooks/useBits'
import { useHeartbeat } from './hooks/useHeartbeat'
import { useClickBoost } from './hooks/useClickBoost'

export default function App() {
  const { context, jwt, products, bitsEnabled, authError } = useTwitch()
  const { buyWithBits, status, error } = useBits(jwt)
  const isViewer = context?.role === 'viewer'
  const { balance, gain, isAnimating, syncBalance } = useHeartbeat(jwt, {
    enabled: isViewer,
  })
  const {
    handleClick,
    cooldownMs,
    isAnimating: clickAnimating,
    gain: clickGain,
  } = useClickBoost(context?.channelId, isViewer, syncBalance)

  if (!context) {
    return (
      <div className="ext-loading">
        <div className="ext-loading__spinner" />
        <span>Connecting…</span>
      </div>
    )
  }

  if (context.role === 'broadcaster') {
    return (
      <div className="ext-panel">
        <header className="ext-header">
          <span className="ext-logo">tachigo</span>
        </header>
        <div className="ext-broadcaster">
          <p>Broadcaster view</p>
          <p className="ext-hint">Viewers can spend Bits to earn rewards.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="ext-panel">
      <header className="ext-header">
        <span className="ext-logo">tachigo</span>
        {authError && <span className="ext-offline" title={authError}>●</span>}
      </header>

      <div className="ext-body">
        <section className="ext-balance-wrap">
          <div className={`ext-balance ${isAnimating ? 'ext-balance--bump' : ''}`}>
            <span className="ext-balance__label">Points</span>
            <strong className="ext-balance__value">{balance?.toLocaleString() ?? '—'}</strong>
          </div>
          {gain !== null && gain > 0 && (
            <span className="ext-balance-gain">+{gain.toLocaleString()} 點</span>
          )}
        </section>

        <section className="ext-mine">
          <div className="ext-mine__wrap">
            <button
              className={`ext-mine__btn${cooldownMs > 0 ? ' ext-mine__btn--cooldown' : ''}`}
              onClick={handleClick}
              disabled={cooldownMs > 0}
              aria-label="Click to mine points"
            >
              ⛏
            </button>
            {clickGain !== null && clickGain > 0 && (
              <span className={`ext-mine__gain${clickAnimating ? '' : ' ext-mine__gain--hidden'}`}>
                +{clickGain}
              </span>
            )}
          </div>
          {cooldownMs > 0 && (
            <span className="ext-mine__cooldown">{(cooldownMs / 1000).toFixed(1)}s</span>
          )}
        </section>

        {status === 'success' && (
          <div className="ext-success">
            <span className="ext-success__icon">✓</span>
            <p>Token received!</p>
            <button className="ext-btn ext-btn--ghost" onClick={() => window.location.reload()}>
              Close
            </button>
          </div>
        )}

        {status !== 'success' && (
          <>
            {bitsEnabled && products.length > 0 ? (
              <ul className="ext-products">
                {products.map((product) => (
                  <li key={product.sku} className="ext-product">
                    <div className="ext-product__info">
                      <span className="ext-product__name">{product.displayName}</span>
                      {product.inDevelopment && (
                        <span className="ext-badge">dev</span>
                      )}
                    </div>
                    <button
                      className="ext-btn ext-btn--bits"
                      disabled={status === 'pending'}
                      onClick={() => buyWithBits(product.sku)}
                    >
                      <img
                        src="https://static-cdn.jtvnw.net/bits/dark/animated/purple/1"
                        alt=""
                        className="ext-bits-icon"
                      />
                      {status === 'pending' ? '…' : product.cost.amount.toLocaleString()}
                    </button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="ext-hint">
                {bitsEnabled ? 'No products available.' : 'Bits not available.'}
              </p>
            )}

            {status === 'error' && (
              <p className="ext-error">{error ?? 'Something went wrong.'}</p>
            )}
          </>
        )}
      </div>
    </div>
  )
}
