import {
  createContext,
  type PropsWithChildren,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';

import { useAuth } from '@/providers/AuthProvider';
import { loadProfile, saveProfile } from '@/services/cache/dollhouse';
import { getUserFacingError } from '@/services/resilience/errors';
import type { Profile, UpdateProfileRequest } from '@/types/api';

type ProfileContextValue = {
  profile: Profile | null;
  isLoading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  update: (request: UpdateProfileRequest) => Promise<Profile>;
};

const ProfileContext = createContext<ProfileContextValue | null>(null);

export function ProfileProvider({ children }: PropsWithChildren) {
  const { isAuthenticated } = useAuth();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!isAuthenticated) return;
    setIsLoading(true);
    setError(null);
    try {
      const result = await loadProfile();
      if (result.cached) setProfile(result.cached.data);
      const refreshed = await result.refresh;
      setProfile(refreshed.data);
    } catch (caught) {
      setError(getUserFacingError(caught, 'Your profile could not be loaded.'));
    } finally {
      setIsLoading(false);
    }
  }, [isAuthenticated]);

  useEffect(() => {
    const timeout = setTimeout(() => {
      if (isAuthenticated) {
        void refresh();
      } else {
        setProfile(null);
        setError(null);
        setIsLoading(false);
      }
    }, 0);
    return () => clearTimeout(timeout);
  }, [isAuthenticated, refresh]);

  const update = useCallback(async (request: UpdateProfileRequest) => {
    const response = await saveProfile(request);
    setProfile(response.data.profile);
    setError(null);
    return response.data.profile;
  }, []);

  const value = useMemo(
    () => ({ profile, isLoading, error, refresh, update }),
    [profile, isLoading, error, refresh, update],
  );

  return (
    <ProfileContext.Provider value={value}>{children}</ProfileContext.Provider>
  );
}

export function useProfile() {
  const context = useContext(ProfileContext);
  if (!context) {
    throw new Error('useProfile must be used inside ProfileProvider.');
  }
  return context;
}
