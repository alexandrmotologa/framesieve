const fs = require('fs');
const path = require('path');
const { Resvg } = require('@resvg/resvg-js');

function buildLogoSvg() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="1024" height="1024">
  <defs>
    <clipPath id="squircle-clip">
      <rect x="24" y="24" width="976" height="976" rx="220" />
    </clipPath>

    <linearGradient id="emerald-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#34d399"/>
      <stop offset="50%" stop-color="#10b981"/>
      <stop offset="100%" stop-color="#059669"/>
    </linearGradient>

    <linearGradient id="cyan-optic" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" stop-color="#00f5ff"/>
      <stop offset="100%" stop-color="#10b981"/>
    </linearGradient>

    <linearGradient id="beak-grad" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" stop-color="#334155"/>
      <stop offset="70%" stop-color="#1e293b"/>
      <stop offset="100%" stop-color="#0f172a"/>
    </linearGradient>

    <filter id="subtle-shadow" x="-10%" y="-10%" width="120%" height="120%">
      <feDropShadow dx="0" dy="16" stdDeviation="20" flood-color="#000000" flood-opacity="0.14" />
    </filter>
  </defs>

  <!-- Luxury White Squircle Container -->
  <rect x="24" y="24" width="976" height="976" rx="220" fill="#ffffff" stroke="#e2e8f0" stroke-width="6" />

  <g clip-path="url(#squircle-clip)">
    <g transform="translate(512, 512)" filter="url(#subtle-shadow)">

      <!-- Hexagonal Architectural Gateway Frame -->
      <polygon points="
        0,-390
        338,-195
        338,195
        0,390
        -338,195
        -338,-195
      " fill="none" stroke="#0f172a" stroke-width="36" stroke-linejoin="round" />

      <!-- Inner Precision Telemetry Ring -->
      <polygon points="
        0,-355
        307,-177
        307,177
        0,355
        -307,177
        -307,-177
      " fill="none" stroke="#10b981" stroke-width="4" opacity="0.45" stroke-dasharray="16, 12" />

      <!-- Corner Telemetry Nodes -->
      <circle cx="0" cy="-390" r="10" fill="#10b981" />
      <circle cx="338" cy="-195" r="10" fill="#10b981" />
      <circle cx="338" cy="195" r="10" fill="#10b981" />
      <circle cx="0" cy="390" r="10" fill="#10b981" />
      <circle cx="-338" cy="195" r="10" fill="#10b981" />
      <circle cx="-338" cy="-195" r="10" fill="#10b981" />

      <!-- 100% Watertight Solid Base Silhouette (Zero-Leak Rule) -->
      <path d="
        M 0 -310
        L 75 -290
        L 140 -230
        L 220 -150
        L 270 -50
        L 260 80
        L 210 200
        L 130 280
        L 0 330
        L -130 280
        L -210 200
        L -260 80
        L -270 -50
        L -220 -150
        L -140 -230
        L -75 -290
        Z
      " fill="#0b0f19" />

      <!-- Crown Crest Facets (Top Head) -->
      <polygon points="0,-310 75,-290 40,-200 0,-215" fill="#334155" />
      <polygon points="0,-310 -75,-290 -40,-200 0,-215" fill="#1e293b" />

      <polygon points="75,-290 140,-230 90,-165 40,-200" fill="#475569" />
      <polygon points="-75,-290 -140,-230 -90,-165 -40,-200" fill="#1e293b" />

      <!-- Forehead & Center Brow -->
      <polygon points="0,-215 40,-200 65,-120 0,-135" fill="#334155" />
      <polygon points="0,-215 -40,-200 -65,-120 0,-135" fill="#0f172a" />

      <!-- Temple & Ear Coverts -->
      <polygon points="140,-230 220,-150 160,-90 90,-165" fill="#334155" />
      <polygon points="-140,-230 -220,-150 -160,-90 -90,-165" fill="#0f172a" />

      <polygon points="220,-150 270,-50 190,-10 160,-90" fill="#475569" />
      <polygon points="-220,-150 -270,-50 -190,-10 -160,-90" fill="#1e293b" />

      <!-- Aggressive Supraorbital Brow Plate -->
      <polygon points="0,-135 65,-120 120,-60 60,-50 0,-70" fill="#475569" />
      <polygon points="0,-135 -65,-120 -120,-60 -60,-50 0,-70" fill="#1e293b" />

      <!-- Raptor Eyes: Geometric Predatory Slit with Electric Cyan-Emerald Optics -->
      <!-- Left Eye Socket & Optic -->
      <polygon points="-120,-60 -60,-50 -65,-25 -115,-30" fill="#022c22" />
      <polygon points="-112,-52 -68,-45 -74,-30 -108,-34" fill="url(#cyan-optic)" />
      <polygon points="-94,-48 -86,-46 -88,-34 -94,-35" fill="#021c16" />

      <!-- Right Eye Socket & Optic -->
      <polygon points="120,-60 60,-50 65,-25 115,-30" fill="#022c22" />
      <polygon points="112,-52 68,-45 74,-30 108,-34" fill="url(#cyan-optic)" />
      <polygon points="94,-48 86,-46 88,-34 94,-35" fill="#021c16" />

      <!-- Cheeks & Malar Facets -->
      <polygon points="160,-90 190,-10 140,60 85,-10" fill="#334155" />
      <polygon points="-160,-90 -190,-10 -140,60 -85,-10" fill="#0f172a" />

      <polygon points="270,-50 260,80 180,110 140,60 190,-10" fill="#1e293b" />
      <polygon points="-270,-50 -260,80 -180,110 -140,60 -190,-10" fill="#0b0f19" />

      <!-- Beak Culmen & Hook (Origami Geometry) -->
      <polygon points="0,-70 60,-50 40,25 0,35" fill="#64748b" />
      <polygon points="0,-70 -60,-50 -40,25 0,35" fill="#334155" />

      <polygon points="0,35 40,25 25,95 0,120" fill="url(#beak-grad)" />
      <polygon points="0,35 -40,25 -25,95 0,120" fill="#0f172a" />

      <!-- Beak Hook Tip Accent -->
      <polygon points="0,120 25,95 0,150" fill="#10b981" opacity="0.9" />
      <polygon points="0,120 -25,95 0,150" fill="#059669" opacity="0.9" />

      <!-- Throat & Gorget Chevrons (Aerodynamic Sieve Plates) -->
      <polygon points="0,150 50,110 90,170 0,225" fill="#334155" />
      <polygon points="0,150 -50,110 -90,170 0,225" fill="#1e293b" />

      <polygon points="90,170 180,110 210,200 130,240" fill="#1e293b" />
      <polygon points="-90,170 -180,110 -210,200 -130,240" fill="#0f172a" />

      <polygon points="0,225 90,170 130,240 0,285" fill="#475569" />
      <polygon points="0,225 -90,170 -130,240 0,285" fill="#1e293b" />

      <!-- Breast Keel Plate (Bottom Apex) -->
      <polygon points="0,285 130,240 75,305 0,330" fill="#334155" />
      <polygon points="0,285 -130,240 -75,305 0,330" fill="#0f172a" />

      <!-- Precision Optical Reticle Accent at Beak Core -->
      <circle cx="0" cy="40" r="4" fill="#00f5ff" />
      <line x1="-12" y1="40" x2="12" y2="40" stroke="#00f5ff" stroke-width="1.5" opacity="0.7" />
      <line x1="0" y1="28" x2="0" y2="52" stroke="#00f5ff" stroke-width="1.5" opacity="0.7" />

    </g>
  </g>
</svg>`;
}

async function main() {
  const svg = buildLogoSvg();
  const outputDir = path.join(__dirname, '..', 'docs', 'images');
  if (!fs.existsSync(outputDir)) {
    fs.mkdirSync(outputDir, { recursive: true });
  }

  const svgPath = path.join(outputDir, 'logo.svg');
  const pngPath = path.join(outputDir, 'logo.png');

  fs.writeFileSync(svgPath, svg, 'utf8');

  const resvg = new Resvg(svg, {
    fitTo: { mode: 'width', value: 1024 },
  });
  const pngData = resvg.render().asPng();
  fs.writeFileSync(pngPath, pngData);

  console.log(`Successfully generated logo at:`);
  console.log(`- SVG: ${svgPath}`);
  console.log(`- PNG: ${pngPath} (1024x1024)`);
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
