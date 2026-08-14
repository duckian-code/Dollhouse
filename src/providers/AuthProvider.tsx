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
  hasAuthenticatedUser,
  login as cognitoLogin,
  logout as cognitoLogout,
} from '@/services/auth/cognito';

type AuthContextValue = {
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    hasAuthenticatedUser()
      .then(setIsAuthenticated)
      .finally(() => setIsLoading(false));
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const result = await cognitoLogin(email, password);
    if (!result.isSignedIn) {
      throw new Error('Additional sign-in verification is required.');
    }
    setIsAuthenticated(true);
  }, []);

  const logout = useCallback(async () => {
    await cognitoLogout();
    setIsAuthenticated(false);
  }, []);

  const value = useMemo(
    () => ({ isAuthenticated, isLoading, login, logout }),
    [isAuthenticated, isLoading, login, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used inside AuthProvider.');
  return context;
}
