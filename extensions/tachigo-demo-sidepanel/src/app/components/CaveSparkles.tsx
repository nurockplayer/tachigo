import React from 'react';

interface SparklePoint {
  x: number;
  y: number;
  dur: number;
  delay: number;
  size: number;
}

const SPARKLES: SparklePoint[] = [
  { x: 12, y: 22, dur: 2.1, delay: 0,    size: 3 },
  { x: 78, y: 12, dur: 2.8, delay: 0.5,  size: 4 },
  { x: 88, y: 30, dur: 1.9, delay: 1.1,  size: 2.5 },
  { x: 22, y: 55, dur: 3.2, delay: 0.3,  size: 3 },
  { x: 93, y: 65, dur: 2.4, delay: 0.9,  size: 2 },
  { x: 55, y: 18, dur: 2.6, delay: 1.6,  size: 3.5 },
  { x: 8,  y: 75, dur: 1.8, delay: 0.7,  size: 2 },
  { x: 68, y: 72, dur: 2.9, delay: 1.4,  size: 2.5 },
  { x: 40, y: 10, dur: 3.5, delay: 2.0,  size: 2 },
  { x: 32, y: 40, dur: 2.3, delay: 0.2,  size: 3 },
];

export function CaveSparkles() {
  return (
    <div
      style={{
        position: 'absolute',
        inset: 0,
        pointerEvents: 'none',
        overflow: 'hidden',
      }}
    >
      {SPARKLES.map((s, i) => (
        <div
          key={i}
          className="sparkle"
          style={
            {
              position: 'absolute',
              left: `${s.x}%`,
              top: `${s.y}%`,
              width: s.size,
              height: s.size,
              borderRadius: '50%',
              background: '#f5c842',
              boxShadow: `0 0 ${s.size * 2}px ${s.size}px rgba(245,200,66,0.6)`,
              '--dur': `${s.dur}s`,
              '--delay': `${s.delay}s`,
            } as React.CSSProperties
          }
        />
      ))}
      {/* Diamond sparkle shapes */}
      {[
        { x: 50, y: 8,  delay: 0.6, size: 5 },
        { x: 18, y: 35, delay: 1.8, size: 4 },
        { x: 82, y: 48, delay: 0.4, size: 4 },
      ].map((d, i) => (
        <div
          key={`d-${i}`}
          className="sparkle"
          style={
            {
              position: 'absolute',
              left: `${d.x}%`,
              top: `${d.y}%`,
              width: d.size,
              height: d.size,
              background: 'rgba(200,200,255,0.7)',
              transform: 'rotate(45deg)',
              boxShadow: `0 0 ${d.size * 2}px rgba(180,200,255,0.5)`,
              '--dur': '2.5s',
              '--delay': `${d.delay}s`,
            } as React.CSSProperties
          }
        />
      ))}
    </div>
  );
}
