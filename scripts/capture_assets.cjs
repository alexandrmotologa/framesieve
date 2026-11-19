const puppeteer = require('puppeteer');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

async function main() {
  const imagesDir = path.join(__dirname, '..', 'docs', 'images');
  const tempFramesDir = path.join(__dirname, '..', 'temp_frames');

  if (!fs.existsSync(imagesDir)) fs.mkdirSync(imagesDir, { recursive: true });
  if (!fs.existsSync(tempFramesDir)) fs.mkdirSync(tempFramesDir, { recursive: true });

  console.log('Launching headless browser...');
  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-gpu'],
  });

  const page = await browser.newPage();

  // Desktop Viewport
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });
  console.log('Navigating to http://localhost:8686...');
  await page.goto('http://localhost:8686', { waitUntil: 'networkidle2', timeout: 30000 });

  console.log('Waiting 4 seconds for live SSE frames and telemetry...');
  await new Promise(r => setTimeout(r, 4000));

  // Screenshot 1: Desktop Dashboard
  const desktopPath = path.join(imagesDir, 'framesieve_dashboard.png');
  await page.screenshot({ path: desktopPath, fullPage: false });
  console.log(`✓ Saved desktop dashboard screenshot: ${desktopPath}`);

  // Recording frames for Animated Demo GIF
  console.log('Capturing animation frames for GIF...');
  const frameCount = 30;
  for (let i = 0; i < frameCount; i++) {
    const framePath = path.join(tempFramesDir, `frame_${String(i).padStart(3, '0')}.png`);
    await page.screenshot({ path: framePath, fullPage: false });
    await new Promise(r => setTimeout(r, 120)); // ~8 FPS recording
  }

  // Mobile Viewport (iPhone 14 Pro: 390x844)
  console.log('Switching to mobile viewport (390x844)...');
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2, isMobile: true });
  await page.reload({ waitUntil: 'networkidle2' });
  await new Promise(r => setTimeout(r, 3000));

  const mobilePath = path.join(imagesDir, 'framesieve_mobile.png');
  await page.screenshot({ path: mobilePath, fullPage: false });
  console.log(`✓ Saved mobile screenshot: ${mobilePath}`);

  await browser.close();

  // Compile GIF with FFmpeg
  console.log('Compiling frames into animated demo GIF using FFmpeg...');
  const gifPath = path.join(imagesDir, 'framesieve_demo.gif');
  const palettePath = path.join(tempFramesDir, 'palette.png');

  try {
    // Generate palette for high quality GIF
    execSync(`ffmpeg -y -i "${path.join(tempFramesDir, 'frame_%03d.png')}" -vf "fps=8,scale=1080:-1:flags=lanczos,palettegen" "${palettePath}"`, { stdio: 'inherit' });
    // Use palette to generate crisp GIF
    execSync(`ffmpeg -y -i "${path.join(tempFramesDir, 'frame_%03d.png')}" -i "${palettePath}" -filter_complex "fps=8,scale=1080:-1:flags=lanczos[x];[x][1:v]paletteuse" "${gifPath}"`, { stdio: 'inherit' });
    console.log(`✓ Successfully generated animated demo GIF: ${gifPath}`);
  } catch (err) {
    console.error('FFmpeg GIF generation failed, falling back to direct conversion:', err);
    execSync(`ffmpeg -y -framerate 8 -i "${path.join(tempFramesDir, 'frame_%03d.png')}" -vf "scale=960:-1" "${gifPath}"`, { stdio: 'inherit' });
  }

  // Cleanup temp frames
  fs.rmSync(tempFramesDir, { recursive: true, force: true });
  console.log('✓ Cleaned up temporary recording frames.');
}

main().catch(err => {
  console.error('Error capturing assets:', err);
  process.exit(1);
});
