type ConnectionState = 'online' | 'offline';
type ConnectionListener = (state: ConnectionState) => void;
type SessionListener = () => void | Promise<void>;

const connectionListeners = new Set<ConnectionListener>();
const sessionListeners = new Set<SessionListener>();
let sessionExpirationInProgress = false;

export function subscribeToConnection(listener: ConnectionListener) {
  connectionListeners.add(listener);
  return () => {
    connectionListeners.delete(listener);
  };
}

export function notifyConnection(state: ConnectionState) {
  connectionListeners.forEach((listener) => listener(state));
}

export function subscribeToSessionExpiration(listener: SessionListener) {
  sessionListeners.add(listener);
  return () => {
    sessionListeners.delete(listener);
  };
}

export async function notifySessionExpired() {
  if (sessionExpirationInProgress) return;
  sessionExpirationInProgress = true;
  try {
    await Promise.all([...sessionListeners].map((listener) => listener()));
  } finally {
    sessionExpirationInProgress = false;
  }
}
