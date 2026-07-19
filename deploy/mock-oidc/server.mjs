import { createHash, createSign, generateKeyPairSync, randomUUID } from 'node:crypto';
import { createServer } from 'node:http';

const issuer = process.env.OIDC_ISSUER ?? 'http://mock-oidc:8080/realms/docs-collab';
const publicBase = process.env.OIDC_PUBLIC_BASE ?? 'http://localhost:8082/realms/docs-collab';
const clientId = process.env.OIDC_CLIENT_ID ?? 'docs-collab';
const clientSecret = process.env.OIDC_CLIENT_SECRET ?? 'docs-collab-secret';
const port = Number.parseInt(process.env.PORT ?? '8080', 10);
const codes = new Map();

const { privateKey, publicKey } = generateKeyPairSync('rsa', { modulusLength: 2048 });
const publicJwk = publicKey.export({ format: 'jwk' });
const kid = createHash('sha256').update(JSON.stringify(publicJwk)).digest('base64url').slice(0, 16);

function json(res, payload, status = 200) {
  res.writeHead(status, { 'content-type': 'application/json; charset=utf-8' });
  res.end(JSON.stringify(payload));
}

function redirect(res, location) {
  res.writeHead(302, { location });
  res.end();
}

function formBody(req) {
  return new Promise((resolve) => {
    let body = '';
    req.on('data', (chunk) => {
      body += chunk;
    });
    req.on('end', () => resolve(new URLSearchParams(body)));
  });
}

function signJWT(payload) {
  const header = { alg: 'RS256', kid, typ: 'JWT' };
  const encode = (value) => Buffer.from(JSON.stringify(value)).toString('base64url');
  const signingInput = `${encode(header)}.${encode(payload)}`;
  const signature = createSign('RSA-SHA256').update(signingInput).sign(privateKey, 'base64url');
  return `${signingInput}.${signature}`;
}

createServer(async (req, res) => {
  const url = new URL(req.url ?? '/', publicBase);
  if (
    url.pathname === '/realms/docs-collab/.well-known/openid-configuration' ||
    url.pathname === '/.well-known/openid-configuration/realms/docs-collab' ||
    url.pathname === '/.well-known/openid-configuration'
  ) {
    json(res, {
      issuer,
      authorization_endpoint: `${publicBase}/protocol/openid-connect/auth`,
      token_endpoint: `${issuer}/protocol/openid-connect/token`,
      userinfo_endpoint: `${issuer}/protocol/openid-connect/userinfo`,
      jwks_uri: `${issuer}/protocol/openid-connect/certs`,
      response_types_supported: ['code'],
      subject_types_supported: ['public'],
      id_token_signing_alg_values_supported: ['RS256'],
      scopes_supported: ['openid', 'email', 'profile'],
      token_endpoint_auth_methods_supported: ['client_secret_post', 'client_secret_basic'],
    });
    return;
  }
  if (url.pathname === '/realms/docs-collab/protocol/openid-connect/certs') {
    json(res, { keys: [{ ...publicJwk, kid, alg: 'RS256', use: 'sig' }] });
    return;
  }
  if (url.pathname === '/realms/docs-collab/protocol/openid-connect/auth') {
    const code = randomUUID();
    codes.set(code, {
      nonce: url.searchParams.get('nonce') ?? '',
      redirectURI: url.searchParams.get('redirect_uri') ?? '',
      clientID: url.searchParams.get('client_id') ?? clientId,
    });
    const redirectURL = new URL(url.searchParams.get('redirect_uri') ?? 'http://localhost:8080/api/auth/oidc/callback');
    redirectURL.searchParams.set('code', code);
    redirectURL.searchParams.set('state', url.searchParams.get('state') ?? '');
    redirect(res, redirectURL.toString());
    return;
  }
  if (url.pathname === '/realms/docs-collab/protocol/openid-connect/token' && req.method === 'POST') {
    const body = await formBody(req);
    const auth = req.headers.authorization ?? '';
    const basic = auth.startsWith('Basic ')
      ? Buffer.from(auth.slice('Basic '.length), 'base64').toString('utf8').split(':')
      : [];
    const requestClientID = body.get('client_id') || basic[0] || '';
    const requestClientSecret = body.get('client_secret') || basic[1] || '';
    const code = body.get('code') ?? '';
    const issued = codes.get(code) ?? (codes.size === 1 ? [...codes.values()][0] : null);
    if (codes.has(code)) {
      codes.delete(code);
    } else if (issued) {
      codes.clear();
    }
    if (!issued || requestClientID !== clientId || requestClientSecret !== clientSecret) {
      json(res, { error: 'invalid_grant' }, 400);
      return;
    }
    const now = Math.floor(Date.now() / 1000);
    const profile = {
      sub: 'mock-oidc-user-1',
      email: 'oidc-user@example.com',
      name: 'OIDC 测试用户',
    };
    json(res, {
      access_token: signJWT({ iss: issuer, aud: clientId, iat: now, exp: now + 3600, ...profile }),
      id_token: signJWT({ iss: issuer, aud: clientId, iat: now, exp: now + 3600, nonce: issued.nonce, ...profile }),
      token_type: 'Bearer',
      expires_in: 3600,
    });
    return;
  }
  if (url.pathname === '/realms/docs-collab/protocol/openid-connect/userinfo') {
    json(res, {
      sub: 'mock-oidc-user-1',
      email: 'oidc-user@example.com',
      name: 'OIDC 测试用户',
    });
    return;
  }
  json(res, { error: 'not found' }, 404);
}).listen(port, '0.0.0.0');
