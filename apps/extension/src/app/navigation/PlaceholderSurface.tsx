import type { ReactNode } from 'react'

interface PlaceholderSurfaceProps {
  title: string
  eyebrow?: string
  body?: string
  action?: ReactNode
  onClick?: () => void
}

export function PlaceholderSurface({ title, eyebrow, body, action, onClick }: PlaceholderSurfaceProps) {
  const content = (
    <>
      {eyebrow ? (
        <div style={{ color: '#7dd3fc', fontSize: 9, letterSpacing: '0.12em', textTransform: 'uppercase' }}>
          {eyebrow}
        </div>
      ) : null}
      <h1 style={{ color: '#f8fafc', fontSize: 24, margin: '18px 0 0', lineHeight: 1.15 }}>{title}</h1>
      {body ? (
        <p style={{ color: 'rgba(226,232,240,0.72)', fontSize: 11, lineHeight: 1.7, margin: '16px 0 0' }}>
          {body}
        </p>
      ) : null}
      {action ? <div style={{ marginTop: 24 }}>{action}</div> : null}
    </>
  )

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        style={{
          appearance: 'none',
          border: 0,
          textAlign: 'left',
          cursor: 'pointer',
          width: '100%',
          height: '100%',
          padding: 28,
          background: 'radial-gradient(circle at 50% 25%, rgba(20,184,166,0.18), transparent 34%), #07111f',
          fontFamily: 'var(--pixel-font-family)',
        }}
      >
        {content}
      </button>
    )
  }

  return (
    <div
      style={{
        width: '100%',
        height: '100%',
        padding: 28,
        boxSizing: 'border-box',
        background: 'radial-gradient(circle at 50% 25%, rgba(20,184,166,0.18), transparent 34%), #07111f',
        fontFamily: 'var(--pixel-font-family)',
      }}
    >
      {content}
    </div>
  )
}
