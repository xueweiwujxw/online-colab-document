import { useEffect, useState } from 'react';

import { getHealth, type HealthResponse } from '../api/health';
import './App.css';

type HealthState =
  | { state: 'loading' }
  | { state: 'success'; data: HealthResponse }
  | { state: 'error'; message: string };

export function App() {
  const [health, setHealth] = useState<HealthState>({ state: 'loading' });

  useEffect(() => {
    let mounted = true;

    getHealth()
      .then((data) => {
        if (mounted) {
          setHealth({ state: 'success', data });
        }
      })
      .catch((error: unknown) => {
        if (mounted) {
          setHealth({
            state: 'error',
            message: error instanceof Error ? error.message : 'Unknown error',
          });
        }
      });

    return () => {
      mounted = false;
    };
  }, []);

  return (
    <main className="shell">
      <section className="status-panel">
        <p className="eyebrow">Online document collaboration</p>
        <h1>Docs Collab Service</h1>
        <div className="health-row">
          <span>Backend health</span>
          <HealthBadge health={health} />
        </div>
      </section>
    </main>
  );
}

function HealthBadge({ health }: { health: HealthState }) {
  if (health.state === 'loading') {
    return <span className="badge badge-loading">Checking</span>;
  }

  if (health.state === 'error') {
    return <span className="badge badge-error">{health.message}</span>;
  }

  return <span className="badge badge-ok">{health.data.status}</span>;
}
