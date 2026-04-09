import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next'
import logoImage from '../../assets/242a2b8162b4542ca6839e84ad45ad4a36c0257c.png';

// ─── Loading Screen (Mario Design Philosophy) ─────────────
export function LoadingScreen({ onComplete }: { onComplete?: () => void }) {
  const { t } = useTranslation()
  const [progress, setProgress] = useState(0);

  // Simulate loading progress
  useEffect(() => {
    const interval = setInterval(() => {
      setProgress(prev => {
        if (prev >= 100) {
          clearInterval(interval);
          setTimeout(() => onComplete?.(), 300);
          return 100;
        }
        return prev + 2;
      });
    }, 50);

    return () => clearInterval(interval);
  }, [onComplete]);

  return (
    <div
      style={{
        width: 320,
        height: 600,
        background: '#0d0d1a',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        position: 'relative',
        overflow: 'hidden',
        fontFamily: 'var(--pixel-font-family)',
        userSelect: 'none',
      }}
    >
      {/* ════════════════════════════════
          CENTERED GROUP (Logo + Loading + Progress)
      ════════════════════════════════ */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 0,
        }}
      >
        {/* 1. LOGO IMAGE */}
        <img
          src={logoImage}
          alt="Tachigo Logo"
          style={{
            width: '35%',
            height: 'auto',
            display: 'block',
            marginBottom: 16,
          }}
        />

        {/* 2. LOADING TEXT */}
        <div
          style={{
            fontFamily: 'var(--pixel-font-family)',
            fontSize: 8,
            color: '#9146FF',
            letterSpacing: '0.1em',
            marginBottom: 8,
          }}
        >
          {t('loading.text')}
        </div>

        {/* 3. PROGRESS BAR */}
        <div
          style={{
            width: 192,
            height: 20,
            background: '#2a2a3e',
            border: '2px solid #9146FF',
            borderRadius: 2,
            overflow: 'hidden',
            position: 'relative',
          }}
        >
          <div
            style={{
              height: '100%',
              width: `${progress}%`,
              background: '#9146FF',
              transition: 'width 0.05s linear',
              boxShadow: '0 0 10px rgba(145,70,255,0.8)',
            }}
          />
        </div>
      </div>
    </div>
  );
}
