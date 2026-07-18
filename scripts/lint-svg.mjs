import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { XMLValidator } from 'fast-xml-parser';

function findSvgFiles(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) return findSvgFiles(path);
    return entry.isFile() && entry.name.endsWith('.svg') ? [path] : [];
  });
}

let failed = false;

for (const file of findSvgFiles('public')) {
  const result = XMLValidator.validate(readFileSync(file, 'utf8'));
  if (result === true) continue;

  failed = true;
  console.error(`${file}:${result.err.line}:${result.err.col}: ${result.err.msg}`);
}

if (failed) process.exitCode = 1;
