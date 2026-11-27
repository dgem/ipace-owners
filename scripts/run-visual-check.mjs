#!/usr/bin/env node

import { createServer } from 'node:http';
import { createReadStream, existsSync, statSync } from 'node:fs';
import { extname, join, resolve, sep } from 'node:path';
import { spawn, spawnSync } from 'node:child_process';

// A supplied URL belongs to the caller; do not start or stop its server.
if (process.env.VISUAL_BASE_URL) {
  const result = spawnSync(process.execPath, ['scripts/visual-check.mjs'], { stdio: 'inherit', env: process.env });
  process.exit(result.status ?? 1);
}

const build = spawnSync('npm', ['run', 'build'], { stdio: 'inherit', env: process.env });
if (build.status !== 0) process.exit(build.status ?? 1);

const site = resolve('_site');
const contentTypes = { '.css': 'text/css', '.html': 'text/html', '.js': 'text/javascript', '.json': 'application/json', '.jpg': 'image/jpeg', '.jpeg': 'image/jpeg', '.png': 'image/png', '.svg': 'image/svg+xml', '.webp': 'image/webp', '.mp4': 'video/mp4', '.woff2': 'font/woff2' };
const server = createServer((request, response) => {
  let pathname;
  try { pathname = decodeURIComponent(new URL(request.url, 'http://localhost').pathname); } catch { response.writeHead(400).end(); return; }
  let file = resolve(site, '.' + pathname);
  if (file !== site && !file.startsWith(site + sep)) { response.writeHead(403).end(); return; }
  if (existsSync(file) && statSync(file).isDirectory()) file = join(file, 'index.html');
  if (!existsSync(file) && /^\/member-stories\/[^/]+\/?$/.test(pathname)) file = join(site, 'member-stories/index.html');
  if (!existsSync(file) || !statSync(file).isFile()) { response.writeHead(404).end(); return; }
  response.writeHead(200, { 'Content-Type': contentTypes[extname(file).toLowerCase()] || 'application/octet-stream' });
  createReadStream(file).pipe(response);
});

server.listen(0, '127.0.0.1', () => {
  const url = `http://127.0.0.1:${server.address().port}`;
  process.stdout.write(`Visual checks using ${url}\n`);
  const child = spawn(process.execPath, ['scripts/visual-check.mjs'], { stdio: 'inherit', env: { ...process.env, VISUAL_BASE_URL: url } });
  const stop = (signal) => child.kill(signal);
  process.once('SIGINT', stop);
  process.once('SIGTERM', stop);
  child.on('error', (error) => { console.error(error.message); server.close(); process.exitCode = 1; });
  child.on('exit', (code) => { server.close(); process.exitCode = code ?? 1; });
});
server.on('error', (error) => { console.error(error.message); process.exitCode = 1; });
