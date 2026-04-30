export const CAVE_SVG_BG = (() => {
  const svg = [
    '<svg xmlns="http://www.w3.org/2000/svg" width="320" height="600">',
    '<path d="M0 0L65 0L42 62L70 128L30 194L60 258L24 322L56 385L16 445L48 500L0 560Z" fill="rgba(20,10,50,0.90)"/>',
    '<path d="M320 0L255 0L278 68L252 136L284 198L256 262L290 324L262 388L298 448L266 502L320 562Z" fill="rgba(20,10,50,0.90)"/>',
    '<path d="M0 0L320 0L320 68L292 44L250 78L205 48L162 70L120 44L82 72L40 42L0 68Z" fill="rgba(6,2,18,0.97)"/>',
    '<path d="M0 600L320 600L320 574L292 582L258 562L224 580L188 562L154 578L118 562L84 580L48 562L12 578Z" fill="rgba(4,1,12,0.95)"/>',
    '<polygon points="2,478 20,456 36,467 42,492 31,512 6,516 0,499" fill="rgba(195,138,20,0.46)"/>',
    '<polygon points="0,516 18,501 29,518 21,540 4,544 0,534" fill="rgba(170,118,14,0.38)"/>',
    '<polygon points="28,454 44,434 58,444 62,464 48,478 30,482" fill="rgba(215,158,30,0.40)"/>',
    '<polygon points="12,482 26,469 36,475 39,494 30,508 14,510" fill="rgba(242,188,48,0.20)"/>',
    '<polygon points="299,474 316,455 320,467 320,491 307,499 292,492" fill="rgba(195,138,20,0.42)"/>',
    '<polygon points="280,495 298,481 316,490 320,514 300,520 278,512" fill="rgba(170,118,14,0.35)"/>',
    '<polygon points="304,452 320,434 320,456 320,468 315,474 304,474" fill="rgba(215,158,30,0.34)"/>',
    '<polygon points="288,498 304,488 316,497 318,514 302,520 284,512" fill="rgba(242,188,48,0.18)"/>',
    '<polygon points="0,204 24,181 40,200 33,228 14,236 0,227" fill="rgba(102,58,210,0.30)"/>',
    '<polygon points="0,248 20,232 32,250 25,271 0,276" fill="rgba(84,46,185,0.24)"/>',
    '<polygon points="6,210 20,196 30,210 26,222 8,226" fill="rgba(155,100,252,0.18)"/>',
    '<polygon points="320,154 302,135 292,156 299,180 320,184" fill="rgba(102,58,210,0.28)"/>',
    '<polygon points="320,196 306,182 299,200 307,220 320,224" fill="rgba(84,46,185,0.22)"/>',
    '<path d="M0 90L58 67L76 90L64 126L40 138L0 126Z" fill="rgba(15,7,40,0.90)"/>',
    '<line x1="25" y1="95" x2="58" y2="112" stroke="rgba(35,16,72,0.70)" stroke-width="1.5"/>',
    '<line x1="38" y1="70" x2="60" y2="84" stroke="rgba(35,16,72,0.55)" stroke-width="1"/>',
    '<path d="M320 100L270 78L256 100L268 128L288 140L320 128Z" fill="rgba(15,7,40,0.90)"/>',
    '<line x1="295" y1="105" x2="268" y2="120" stroke="rgba(35,16,72,0.65)" stroke-width="1.5"/>',
    '</svg>',
  ].join('')
  return `url("data:image/svg+xml,${encodeURIComponent(svg)}")`
})()

export const hudPanelBackground = [
  'radial-gradient(ellipse 54% 44% at 50% 42%, #0d0d1a 28%, transparent 100%)',
  `${CAVE_SVG_BG} center/100% 100% no-repeat`,
  'radial-gradient(ellipse 48% 65% at -8% 62%, rgba(82,34,152,0.42) 0%, transparent 55%)',
  'radial-gradient(ellipse 44% 60% at 108% 38%, rgba(65,27,135,0.36) 0%, transparent 55%)',
  'radial-gradient(ellipse 95% 32% at 50% -8%, rgba(4,2,12,0.95) 0%, transparent 78%)',
  'radial-gradient(ellipse 100% 28% at 50% 108%, rgba(10,4,25,0.82) 0%, transparent 70%)',
  'radial-gradient(ellipse 24% 15% at 4% 84%, rgba(198,148,20,0.30) 0%, transparent 62%)',
  'radial-gradient(ellipse 20% 12% at 96% 80%, rgba(178,128,16,0.24) 0%, transparent 62%)',
  'radial-gradient(ellipse 130% 85% at 50% 30%, rgba(28,12,68,0.30) 0%, transparent 70%)',
  'linear-gradient(170deg, #0e0d22 0%, #060510 45%, #0a0818 100%)',
].join(', ')
