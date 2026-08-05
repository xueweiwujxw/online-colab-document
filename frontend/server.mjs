import { createReadStream } from 'node:fs';
import { stat } from 'node:fs/promises';
import { extname, join, normalize } from 'node:path';
import { request, createServer } from 'node:http';

const root = '/app/dist';
const port = Number.parseInt(process.env.PORT ?? '3000', 10);
const apiProxyTarget = new URL(process.env.API_PROXY_TARGET ?? 'http://backend:8080');
const officeCollabProxyHost = process.env.OFFICE_COLLAB_PROXY_HOST ?? 'office-collab';
const officeCollabProxyPort = Number.parseInt(process.env.OFFICE_COLLAB_PROXY_PORT ?? '1234', 10);

const contentTypes = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml',
};

const server = createServer(async (req, res) => {
  const path = normalize(decodeURIComponent(req.url?.split('?')[0] ?? '/'));
  if (path.startsWith('/api/')) {
    proxyAPI(req, res);
    return;
  }

  const relativePath = path === '/' ? '/index.html' : path;
  const filePath = join(root, relativePath);

  if (!filePath.startsWith(root)) {
    res.writeHead(403);
    res.end('Forbidden');
    return;
  }

  try {
    const file = await stat(filePath);
    if (!file.isFile()) {
      throw new Error('not a file');
    }

    res.writeHead(200, {
      'Content-Type': contentTypes[extname(filePath)] ?? 'application/octet-stream',
    });
    createReadStream(filePath).pipe(res);
  } catch {
    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    createReadStream(join(root, 'index.html')).pipe(res);
  }
});

server.on('upgrade', (req, socket, head) => {
  const rawPath = req.url ?? '/';
  if (rawPath.startsWith('/api/')) {
    proxyWebSocket(req, socket, head, {
      hostname: apiProxyTarget.hostname,
      port: apiProxyTarget.port || 80,
      host: apiProxyTarget.host,
      path: rawPath,
    });
    return;
  }
  if (!rawPath.startsWith('/office-collab')) {
    socket.destroy();
    return;
  }

  const upstreamPath = rawPath.slice('/office-collab'.length) || '/';

  proxyWebSocket(req, socket, head, {
    hostname: officeCollabProxyHost,
    port: officeCollabProxyPort,
    host: `${officeCollabProxyHost}:${officeCollabProxyPort}`,
    path: upstreamPath.startsWith('/') ? upstreamPath : `/${upstreamPath}`,
  });
});

function proxyWebSocket(req, socket, head, target) {
  const proxyReq = request({
    hostname: target.hostname,
    port: target.port,
    method: req.method,
    path: target.path,
    headers: { ...req.headers, host: target.host },
  });
  proxyReq.on('upgrade', (proxyRes, upstream, upstreamHead) => {
    const headers = [`HTTP/${proxyRes.httpVersion} ${proxyRes.statusCode} ${proxyRes.statusMessage}`];
    for (const [name, value] of Object.entries(proxyRes.headers)) {
      if (value !== undefined) {
        headers.push(`${name}: ${Array.isArray(value) ? value.join(', ') : value}`);
      }
    }
    socket.write(`${headers.join('\r\n')}\r\n\r\n`);
    if (upstreamHead.length > 0) {
      socket.write(upstreamHead);
    }
    if (head.length > 0) {
      upstream.write(head);
    }
    upstream.pipe(socket);
    socket.pipe(upstream);
  });
  proxyReq.on('response', (proxyRes) => {
    socket.write(`HTTP/${proxyRes.httpVersion} ${proxyRes.statusCode} ${proxyRes.statusMessage}\r\n\r\n`);
    socket.destroy();
  });
  proxyReq.on('error', (error) => {
    console.error(`office collab proxy error: ${error.code ?? error.message}`);
    socket.destroy();
  });
  proxyReq.end();
}

server.listen(port, '0.0.0.0');

function proxyAPI(req, res) {
  const proxyReq = request(
    {
      hostname: apiProxyTarget.hostname,
      port: apiProxyTarget.port || 80,
      protocol: apiProxyTarget.protocol,
      method: req.method,
      path: req.url,
      headers: {
        ...req.headers,
        host: apiProxyTarget.host,
      },
    },
    (proxyRes) => {
      res.writeHead(proxyRes.statusCode ?? 502, proxyRes.headers);
      proxyRes.pipe(res);
    },
  );
  proxyReq.on('error', () => {
    res.writeHead(502, { 'Content-Type': 'text/plain; charset=utf-8' });
    res.end('Bad Gateway');
  });
  req.pipe(proxyReq);
}
