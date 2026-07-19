import { Server } from '@hocuspocus/server';

const port = Number.parseInt(process.env.PORT ?? '1234', 10);
const backendURL = trimRight(process.env.BACKEND_INTERNAL_URL ?? 'http://backend:8080', '/');
const sessionPathPrefix = '/api/documents/';

function trimRight(value, suffix) {
  let result = value;
  while (result.endsWith(suffix)) {
    result = result.slice(0, -suffix.length);
  }
  return result;
}

function documentIDFromRoom(room) {
  if (!room.startsWith('office:')) {
    return null;
  }
  const documentID = room.slice('office:'.length);
  return documentID === '' ? null : documentID;
}

async function authorize({ documentName, requestHeaders, requestParameters }) {
  const documentID = documentIDFromRoom(documentName);
  if (!documentID) {
    throw new Error('invalid office collab room');
  }

  const requestedRole = requestParameters.get('role') === 'view' ? 'view' : 'write';
  const cookie = requestHeaders.get('cookie') ?? '';
  if (!cookie) {
    throw new Error('missing session cookie');
  }

  const response = await fetch(
    `${backendURL}${sessionPathPrefix}${encodeURIComponent(documentID)}/office/collab/session`,
    {
      headers: {
        accept: 'application/json',
        cookie,
      },
    },
  );
  if (!response.ok) {
    throw new Error(`office collab authorization failed: ${response.status}`);
  }

  const session = await response.json();
  if (session.fileExt !== 'xlsx' || session.room !== documentName) {
    throw new Error('office collab session mismatch');
  }
  if (requestedRole === 'write' && session.role !== 'write') {
    throw new Error('office collab write permission denied');
  }
  return {
    documentID,
    role: requestedRole,
    userRole: session.role,
  };
}

const server = new Server({
  address: '0.0.0.0',
  port,
  timeout: 30000,
  debounce: 1000,
  maxDebounce: 5000,
  async onAuthenticate(data) {
    return authorize(data);
  },
  async onConnect(data) {
    await authorize(data);
  },
});

server.listen();

console.log(`office collab service listening on :${port}`);
