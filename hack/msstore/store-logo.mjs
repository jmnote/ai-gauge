import fs from "node:fs";
import path from "node:path";
import { repoRoot } from "../lib/paths.mjs";
import { ensureImport } from "../lib/ensure-npm.mjs";
import { isMain } from "../lib/is-main.mjs";

// Microsoft Store display images (Partner Center > Store listing > Store logos),
// uploaded by hand: 9:16 poster art (also the default Windows logo and required
// on Xbox) and 1:1 box art, each at the larger accepted size.
const STORE_ART = [
  { name: "poster-art-1440x2160.png", width: 1440, height: 2160 },
  { name: "box-art-2160x2160.png", width: 2160, height: 2160 },
];

// The gauge in logo.svg only spans this fraction of its square canvas
// vertically, so layout is computed from the visible gauge, not the canvas.
const LOGO_VISIBLE_TOP = 0.09;
const LOGO_VISIBLE_BOTTOM = 0.72;

function storeArtSvg(logoSvg, width, height) {
  const logoSize = Math.round(width * 0.55);
  const fontSize = Math.round(width * 0.09);
  const gap = Math.round(width * 0.07);
  const gaugeHeight = logoSize * (LOGO_VISIBLE_BOTTOM - LOGO_VISIBLE_TOP);
  const blockTop = (height - (gaugeHeight + gap + fontSize)) / 2;
  const logoX = (width - logoSize) / 2;
  const logoY = blockTop - logoSize * LOGO_VISIBLE_TOP;
  const textY = blockTop + gaugeHeight + gap + fontSize * 0.75;
  const logoHref = `data:image/svg+xml;base64,${logoSvg.toString("base64")}`;
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">
  <defs>
    <radialGradient id="glow" cx="50%" cy="45%" r="60%">
      <stop offset="0" stop-color="#2a2d36"/>
      <stop offset="1" stop-color="#16171b"/>
    </radialGradient>
  </defs>
  <rect width="${width}" height="${height}" fill="url(#glow)"/>
  <image href="${logoHref}" x="${logoX}" y="${logoY}" width="${logoSize}" height="${logoSize}"/>
  <text x="${width / 2}" y="${textY}" text-anchor="middle" font-family="Segoe UI" font-weight="600" font-size="${fontSize}" fill="#f4f4f5">AI Gauge</text>
</svg>`;
}

async function generateStoreArt() {
  const logoSvg = fs.readFileSync(path.join(repoRoot, "frontend", "images", "logo.svg"));
  const outputDir = path.join(repoRoot, "dist", "store");
  fs.mkdirSync(outputDir, { recursive: true });
  const { Resvg } = await ensureImport("@resvg/resvg-js", "Store logo generation");
  for (const art of STORE_ART) {
    const svg = storeArtSvg(logoSvg, art.width, art.height);
    const png = new Resvg(svg, {
      fitTo: { mode: "original" },
      font: { loadSystemFonts: true, defaultFontFamily: "Segoe UI" },
    }).render().asPng();
    const output = path.join(outputDir, art.name);
    fs.writeFileSync(output, png);
    console.log(`Created: ${path.relative(repoRoot, output)}`);
  }
}

if (isMain(import.meta.url)) {
  generateStoreArt().catch((err) => {
    console.error(err.message || err);
    process.exit(1);
  });
}
