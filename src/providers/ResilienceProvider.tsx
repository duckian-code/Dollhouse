import {
  createContext,
  type PropsWithChildren,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { Platform } from 'react-native';

import { subscribeToConnection } from '@/services/resilience/events';

interface ResilienceContextValue {
  isOffline: boolean;
  setOfflineForPreview: (offline: boolean) => void;
}

const ResilienceContext = createContext<ResilienceContextValue | null>(null);

export function ResilienceProvider({ children }: PropsWithChildren) {
  const [isOffline, setIsOffline] = useState(false);

  useEffect(
    () => subscribeToConnection((state) => setIsOffline(state === 'offline')),
    [],
  );

  useEffect(() => {
    if (Platform.OS !== 'web' || !globalThis.addEventListener) return;
    const handleOnline = () => setIsOffline(false);
    const handleOffline = () => setIsOffline(true);
    globalThis.addEventListener('online', handleOnline);
    globalThis.addEventListener('offline', handleOffline);
    return () => {
      globalThis.removeEventListener('online', handleOnline);
      globalThis.removeEventListener('offline', handleOffline);
    };
  }, []);

  const value = useMemo(
    () => ({ isOffline, setOfflineForPreview: setIsOffline }),
    [isOffline],
  );

  return (
    <ResilienceContext.Provider value={value}>
      {children}
    </ResilienceContext.Provider>
  );
}

export function useResilience() {
  const context = useContext(ResilienceContext);
  if (!context)
    throw new Error('useResilience must be used inside ResilienceProvider.');
  return context;
}
