import {
  createContext,
  type PropsWithChildren,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';

import {
  getCurrentAccountId,
  hasAuthenticatedUser,
  login as cognitoLogin,
  logout as cognitoLogout,
} from '@/services/auth/cognito';
import {
  clearAccountCache,
  initializeCache,
  pruneExpiredCache,
} from '@/services/cache/repositories';
import { subscribeToSessionExpiration } from '@/services/resilience/events';

type AuthContextValue = {
  isAuthenticated: boolean;
  isLoading: boolean;
  sessionMessage: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [sessionMessage, setSessionMessage] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([initializeCache(), hasAuthenticatedUser()])
      .then(([, authenticated]) => {
        setIsAuthenticated(authenticated);
        void pruneExpiredCache();
      })
      .finally(() => setIsLoading(false));
  }, []);

  useEffect(
    () =>
      subscribeToSessionExpiration(async () => {
        try {
          const accountId = await getCurrentAccountId();
          await clearAccountCache(accountId);
        } catch {
          // An expired token can also make account lookup unavailable.
        }
        try {
          await cognitoLogout();
        } catch {
          // Local navigation should still recover even if remote sign-out fails.
        }
        setIsAuthenticated(false);
        setSessionMessage('Your session expired. Please sign in again.');
      }),
    [],
  );

  const login = useCallback(async (email: string, password: string) => {
    const result = await cognitoLogin(email, password);
    if (!result.isSignedIn) {
      throw new Error('Additional sign-in verification is required.');
    }
    setIsAuthenticated(true);
    setSessionMessage(null);
  }, []);

  const logout = useCallback(async () => {
    const accountId = await getCurrentAccountId();
    await clearAccountCache(accountId);
    await cognitoLogout();
    setIsAuthenticated(false);
  }, []);

  const value = useMemo(
    () => ({ isAuthenticated, isLoading, sessionMessage, login, logout }),
    [isAuthenticated, isLoading, sessionMessage, login, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used inside AuthProvider.');
  return context;
}
