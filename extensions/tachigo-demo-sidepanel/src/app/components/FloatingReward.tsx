interface FloatItem {
  id: number;
  amount: number;
  x: number; // percentage 0-100
}

interface Props {
  items: FloatItem[];
}

export function FloatingReward({ items }: Props) {
  return (
    <>
      {items.map((item) => (
        <div
          key={item.id}
          className="float-reward"
          style={{
            position: 'absolute',
            left: `${item.x}%`,
            bottom: '30%',
            transform: 'translateX(-50%)',
            pointerEvents: 'none',
            zIndex: 30,
            fontFamily: "'Inter', sans-serif",
            fontWeight: 700,
            fontSize: 15,
            color: '#f5c842',
            textShadow: '0 0 8px rgba(245,200,66,0.8), 0 1px 2px rgba(0,0,0,0.8)',
            whiteSpace: 'nowrap',
            letterSpacing: '0.02em',
          }}
        >
          +{item.amount} 點
        </div>
      ))}
    </>
  );
}

export type { FloatItem };
