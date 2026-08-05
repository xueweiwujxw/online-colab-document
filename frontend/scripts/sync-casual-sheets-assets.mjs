import { copyFileSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = dirname(dirname(fileURLToPath(import.meta.url)));
copyEmbedAssets('sheets', 'casual-sheets', [
  'embed.html',
  'embed-runtime.js',
  'exporter.worker.js',
  'formula.worker.js',
  'parser.worker.js',
]);
copyFileSync(join(root, 'scripts', 'casual-sheets-zh-CN.js'), join(root, 'public', 'casual-sheets', 'zh-CN.js'));
const sheetsEmbedPath = join(root, 'public', 'casual-sheets', 'embed.html');
writeFileSync(
  sheetsEmbedPath,
  readFileSync(sheetsEmbedPath, 'utf8')
    .replace('<html lang="en">', '<html lang="zh-CN">')
    .replace('</body>', '    <script src="./zh-CN.js"></script>\n  </body>'),
);
copyEmbedAssets('docs', 'casual-docs', [
  'embed.html',
  { from: 'embed-runtime.mjs', to: 'embed-runtime.js' },
  'embed-runtime.css',
]);
copyDocsFonts();

function copyEmbedAssets(packageName, publicDir, filenames) {
  const sourceDir = join(root, 'node_modules', '@casualoffice', packageName, 'dist', 'embed');
  const targetDir = join(root, 'public', publicDir);
  mkdirSync(targetDir, { recursive: true });
  for (const entry of filenames) {
    const from = typeof entry === 'string' ? entry : entry.from;
    const to = typeof entry === 'string' ? entry : entry.to;
    copyFileSync(join(sourceDir, from), join(targetDir, to));
  }
}

function copyDocsFonts() {
  const sourceDir = join(root, 'node_modules', '@casualoffice', 'docs', 'dist', 'fonts');
  const targetDir = join(root, 'public', 'fonts');
  mkdirSync(targetDir, { recursive: true });
  for (const filename of readdirSync(sourceDir)) {
    copyFileSync(join(sourceDir, filename), join(targetDir, filename));
  }
}
