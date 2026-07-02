import { useEffect, useState } from 'react'

import { getCurrentAccount, type CurrentAccount } from '../../services/api'

interface AccountPanelProps {
  onBack: () => void
}

type AccountState =
  | { status: 'loading'; account?: undefined; error?: undefined }
  | { status: 'success'; account: CurrentAccount; error?: undefined }
  | { status: 'error'; account?: undefined; error: string }

const panelStyle = {
  width: 320,
  height: 600,
  display: 'flex',
  flexDirection: 'column',
  color: '#e0f2fe',
  background: '#08111f',
  fontFamily: 'var(--pixel-font-family)',
  overflow: 'hidden',
  position: 'relative',
} as const

const backButtonStyle = {
  background: 'none',
  border: 'none',
  color: '#38bdf8',
  cursor: 'pointer',
  fontSize: 8,
  letterSpacing: 0,
  padding: 0,
  fontFamily: 'var(--pixel-font-family)',
} as const

const detailCardStyle = {
  border: '1px solid rgba(56,189,248,0.24)',
  borderRadius: 8,
  padding: '12px 14px',
  background: 'rgba(15,23,42,0.72)',
  display: 'grid',
  gap: 6,
} as const

function formatAccountName(account: CurrentAccount) {
  return account.username ?? account.email ?? account.id
}

function formatActiveStatus(account: CurrentAccount) {
  if (account.isActive === null) {
    return null
  }

  return account.isActive ? 'Active' : 'Inactive'
}

function formatEmailVerified(account: CurrentAccount) {
  if (account.emailVerified === null) {
    return null
  }

  return account.emailVerified ? 'Email verified' : 'Email not verified'
}

export function AccountPanel({ onBack }: AccountPanelProps) {
  const [state, setState] = useState<AccountState>({ status: 'loading' })

  useEffect(() => {
    let isMounted = true

    getCurrentAccount()
      .then((account) => {
        if (isMounted) {
          setState({ status: 'success', account })
        }
      })
      .catch(() => {
        if (isMounted) {
          setState({ status: 'error', error: 'Could not load account' })
        }
      })

    return () => {
      isMounted = false
    }
  }, [])

  return (
    <div style={panelStyle}>
      <div
        style={{
          padding: '14px 16px 12px',
          borderBottom: '1px solid rgba(56,189,248,0.18)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <button type="button" onClick={onBack} style={backButtonStyle}>
          Back
        </button>
        <div style={{ fontSize: 8, color: '#7dd3fc', letterSpacing: 0 }}>ACCOUNT</div>
      </div>

      <div style={{ padding: 16, display: 'grid', gap: 12 }}>
        {state.status === 'loading' ? (
          <div role="status" aria-live="polite" style={detailCardStyle}>
            Loading account
          </div>
        ) : state.status === 'error' ? (
          <div style={{ display: 'grid', gap: 12 }}>
            <div role="alert" style={detailCardStyle}>
              {state.error}
            </div>
            <button type="button" onClick={onBack} style={{ ...backButtonStyle, justifySelf: 'start' }}>
              Close
            </button>
          </div>
        ) : (
          <>
            <section style={{ ...detailCardStyle, gap: 10 }}>
              <div style={{ fontSize: 7, color: '#7dd3fc', letterSpacing: 0 }}>PROFILE</div>
              <div style={{ color: '#f8fafc', fontSize: 18, lineHeight: 1.2 }}>{formatAccountName(state.account)}</div>
              {state.account.email && (
                <div style={{ color: '#bae6fd', fontSize: 8, lineHeight: 1.5 }}>{state.account.email}</div>
              )}
            </section>

            <section style={detailCardStyle}>
              <div style={{ fontSize: 7, color: '#7dd3fc', letterSpacing: 0 }}>ACCOUNT ROLE</div>
              <div style={{ color: '#f8fafc', fontSize: 14 }}>{state.account.role}</div>
            </section>

            {(formatActiveStatus(state.account) || formatEmailVerified(state.account)) && (
              <section style={detailCardStyle}>
                <div style={{ fontSize: 7, color: '#7dd3fc', letterSpacing: 0 }}>STATUS</div>
                {formatActiveStatus(state.account) && (
                  <div style={{ color: '#f8fafc', fontSize: 10 }}>{formatActiveStatus(state.account)}</div>
                )}
                {formatEmailVerified(state.account) && (
                  <div style={{ color: '#f8fafc', fontSize: 10 }}>{formatEmailVerified(state.account)}</div>
                )}
              </section>
            )}
          </>
        )}
      </div>
    </div>
  )
}
