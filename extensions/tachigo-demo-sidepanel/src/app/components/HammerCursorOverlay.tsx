import { useEffect, useRef, type RefObject } from "react";

interface HammerCursorOverlayProps {
  containerRef: RefObject<HTMLElement | null>;
}

export function HammerCursorOverlay({
  containerRef,
}: HammerCursorOverlayProps) {
  const hammerRef = useRef<HTMLDivElement | null>(null);
  const rafRef = useRef<number | null>(null);
  const latestPointRef = useRef<{ x: number; y: number } | null>(null);
  const isPressedRef = useRef(false);

  useEffect(() => {
    const hammer = hammerRef.current;
    const container = containerRef.current;
    if (!hammer || !container) {
      return;
    }

    const render = () => {
      rafRef.current = null;
      const point = latestPointRef.current;
      if (!point) {
        return;
      }

      const rotate = isPressedRef.current ? -48 : -24;
      const scale = isPressedRef.current ? 0.94 : 1;
      hammer.style.transform = `translate(${point.x}px, ${point.y}px) rotate(${rotate}deg) scale(${scale})`;
    };

    const scheduleRender = () => {
      if (rafRef.current !== null) {
        return;
      }
      rafRef.current = window.requestAnimationFrame(render);
    };

    const handlePointerMove = (event: PointerEvent) => {
      const nextContainer = containerRef.current;
      const nextHammer = hammerRef.current;
      if (!nextContainer || !nextHammer) {
        return;
      }

      const rect = nextContainer.getBoundingClientRect();
      const inside =
        event.clientX >= rect.left &&
        event.clientX <= rect.right &&
        event.clientY >= rect.top &&
        event.clientY <= rect.bottom;

      if (!inside) {
        nextHammer.style.opacity = "0";
        return;
      }

      latestPointRef.current = {
        x: event.clientX - rect.left - 22,
        y: event.clientY - rect.top - 18,
      };
      nextHammer.style.opacity = "1";
      scheduleRender();
    };

    const handlePointerDown = () => {
      isPressedRef.current = true;
      scheduleRender();
    };

    const handlePointerUp = () => {
      isPressedRef.current = false;
      scheduleRender();
    };

    const hideHammer = () => {
      const nextHammer = hammerRef.current;
      if (!nextHammer) {
        return;
      }
      nextHammer.style.opacity = "0";
      isPressedRef.current = false;
    };

    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerdown", handlePointerDown);
    window.addEventListener("pointerup", handlePointerUp);
    window.addEventListener("pointerleave", hideHammer);
    window.addEventListener("blur", hideHammer);

    return () => {
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerdown", handlePointerDown);
      window.removeEventListener("pointerup", handlePointerUp);
      window.removeEventListener("pointerleave", hideHammer);
      window.removeEventListener("blur", hideHammer);
      if (rafRef.current !== null) {
        window.cancelAnimationFrame(rafRef.current);
      }
    };
  }, [containerRef]);

  return (
    <div
      ref={hammerRef}
      aria-hidden
      className="hammer-cursor-overlay"
      style={{ opacity: 0 }}
    >
      <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
        <g filter="url(#hammerShadow)">
          <rect x="25" y="14" width="10" height="24" rx="5" fill="#8B6038" />
          <rect x="26" y="15" width="4" height="22" rx="2" fill="#A87448" fillOpacity="0.5" />
          <path d="M10 9L25 14V19L10 22Z" fill="#7A8B9A" />
          <path d="M35 14L40 11L43 6L45 11L35 19Z" fill="#7A8B9A" />
          <path d="M13 11L23 15V17L13 15Z" fill="#A8C0D0" fillOpacity="0.6" />
          <path d="M37 15L43 10L43.8 11.6L37 17Z" fill="#A8C0D0" fillOpacity="0.6" />
        </g>
        <defs>
          <filter
            id="hammerShadow"
            x="6"
            y="4"
            width="42"
            height="42"
            filterUnits="userSpaceOnUse"
            colorInterpolationFilters="sRGB"
          >
            <feFlood floodOpacity="0" result="BackgroundImageFix" />
            <feColorMatrix
              in="SourceAlpha"
              type="matrix"
              values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
              result="hardAlpha"
            />
            <feOffset dy="2" />
            <feGaussianBlur stdDeviation="2" />
            <feColorMatrix
              type="matrix"
              values="0 0 0 0 0.0196078 0 0 0 0 0.0117647 0 0 0 0 0.0627451 0 0 0 0.45 0"
            />
            <feBlend
              mode="normal"
              in2="BackgroundImageFix"
              result="effect1_dropShadow_1_1"
            />
            <feBlend
              mode="normal"
              in="SourceGraphic"
              in2="effect1_dropShadow_1_1"
              result="shape"
            />
          </filter>
        </defs>
      </svg>
    </div>
  );
}
