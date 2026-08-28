import {
  createContext,
  type PropsWithChildren,
  useCallback,
  useContext,
  useMemo,
  useState,
} from 'react';

import type { MoodState } from '@/types/api';

interface MoodStatusContextValue {
  currentStatus: MoodState | null;
  refreshCurrentStatus: (status: MoodState) => void;
}

const MoodStatusContext = createContext<MoodStatusContextValue | null>(null);

export function MoodStatusProvider({ children }: PropsWithChildren) {
  const [currentStatus, setCurrentStatus] = useState<MoodState | null>(null);
  const refreshCurrentStatus = useCallback((status: MoodState) => {
    setCurrentStatus(status);
  }, []);
  const value = useMemo(
    () => ({ currentStatus, refreshCurrentStatus }),
    [currentStatus, refreshCurrentStatus],
  );

  return (
    <MoodStatusContext.Provider value={value}>
      {children}
    </MoodStatusContext.Provider>
  );
}

export function useMoodStatus() {
  const context = useContext(MoodStatusContext);
  if (!context) {
    throw new Error('useMoodStatus must be used inside MoodStatusProvider.');
  }
  return context;
}
