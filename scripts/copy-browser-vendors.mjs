import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const copies = [
  ['node_modules/pdfjs-dist/legacy/build/pdf.min.mjs', 'src/assets/vendor/pdfjs/pdf.min.mjs'],
  ['node_modules/pdfjs-dist/legacy/build/pdf.worker.min.mjs', 'src/assets/vendor/pdfjs/pdf.worker.min.mjs'],
  ['node_modules/tesseract.js/dist/tesseract.min.js', 'src/assets/vendor/tesseract/tesseract.min.js'],
  ['node_modules/tesseract.js/dist/worker.min.js', 'src/assets/vendor/tesseract/worker.min.js'],
  ['node_modules/@tesseract.js-data/eng/4.0.0/eng.traineddata.gz', 'src/assets/vendor/tessdata/4.0.0/eng.traineddata.gz']
];

const coreDirectory = path.join(root, 'node_modules/tesseract.js-core');
for (const name of fs.readdirSync(coreDirectory)) {
  if (name.startsWith('tesseract-core') && name.endsWith('.js')) copies.push(['node_modules/tesseract.js-core/' + name, 'src/assets/vendor/tesseract-core/' + name]);
}
for (const [from, to] of copies) {
  const destination = path.join(root, to);
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  fs.copyFileSync(path.join(root, from), destination);
}
