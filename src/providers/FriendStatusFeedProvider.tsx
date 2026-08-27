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
import { developmentAvatarCatalog } from '@/services/avatar/catalog';
import {
  loadAvatarCatalog,
  loadRecentStatuses,
} from '@/services/cache/dollhouse';
import type { AvatarCatalog, FriendStatus } from '@/types/api';

const previewStatuses: FriendStatus[] = [
  {
    friend: {
      userId: 'preview-riley',
      username: 'rileyk',
      displayName: 'Riley Kim',
    },
    doll: {
      bodyAssetId: 'body-00',
      hairAssetId: 'hair-00',
      eyesAssetId: 'eyes-00',
      noseAssetId: 'nose-00',
      mouthAssetId: 'mouth-00',
      clothingAssetIds: [],
      updatedAt: '2026-08-25T18:00:00Z',
    },
    status: {
      status: 'Calm',
      stress: 2,
      fatigue: 4,
      discomfort: null,
      updatedAt: new Date(Date.now() - 12 * 60 * 1000).toISOString(),
    },
  },
  {
    friend: {
      userId: 'preview-alex',
      username: 'alexj',
      displayName: 'Alex Johnson',
    },
    doll: {
      bodyAssetId: 'body-00',
      hairAssetId: 'hair-02',
      eyesAssetId: 'eyes-00',
      noseAssetId: 'nose-00',
      mouthAssetId: 'mouth-00',
      clothingAssetIds: [],
      updatedAt: '2026-08-24T15:00:00Z',
    },
    status: null,
  },
];

interface FriendStatusFeedContextValue {
  statuses: FriendStatus[];
  catalog: AvatarCatalog | null;
  isInitialLoading: boolean;
  isRefreshing: boolean;
  error: string | null;
  notice: string | null;
  isPreview: boolean;
  refresh: () => Promise<void>;
}

const FriendStatusFeedContext =
  createContext<FriendStatusFeedContextValue | null>(null);

function errorMessage(caught: unknown) {
  return caught instanceof Error
    ? caught.message
    : 'Friend statuses could not be refreshed.';
}

export function FriendStatusFeedProvider({ children }: PropsWithChildren) {
  const { isAuthenticated } = useAuth();
  const isPreview = __DEV__ && !isAuthenticated;
  const [statuses, setStatuses] = useState<FriendStatus[]>(
    isPreview ? previewStatuses : [],
  );
  const [catalog, setCatalog] = useState<AvatarCatalog | null>(
    isPreview ? developmentAvatarCatalog : null,
  );
  const [isInitialLoading, setIsInitialLoading] = useState(!isPreview);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (isPreview) {
      setStatuses(previewStatuses);
      setCatalog(developmentAvatarCatalog);
      setNotice('Local preview data refreshed.');
      return;
    }

    setIsRefreshing(true);
    setError(null);
    setNotice(null);
    try {
      const [statusLoad, catalogLoad] = await Promise.all([
        loadRecentStatuses(),
        loadAvatarCatalog(),
      ]);

      if (statusLoad.cached) {
        setStatuses(statusLoad.cached.data);
        if (statusLoad.cached.isStale) {
          setNotice('Showing saved statuses while checking for updates.');
        }
      }
      if (catalogLoad.cached) setCatalog(catalogLoad.cached.data);

      const [freshStatuses, freshCatalog] = await Promise.all([
        statusLoad.refresh,
        catalogLoad.refresh,
      ]);
      setStatuses(freshStatuses.data);
      setCatalog(freshCatalog.data);
      setNotice(null);
    } catch (caught) {
      setError(errorMessage(caught));
    } finally {
      setIsInitialLoading(false);
      setIsRefreshing(false);
    }
  }, [isPreview]);

  useEffect(() => {
    if (isPreview) return;
    const timeout = setTimeout(() => void refresh(), 0);
    return () => clearTimeout(timeout);
  }, [isPreview, refresh]);

  const value = useMemo(
    () => ({
      statuses,
      catalog,
      isInitialLoading,
      isRefreshing,
      error,
      notice,
      isPreview,
      refresh,
    }),
    [
      statuses,
      catalog,
      isInitialLoading,
      isRefreshing,
      error,
      notice,
      isPreview,
      refresh,
    ],
  );

  return (
    <FriendStatusFeedContext.Provider value={value}>
      {children}
    </FriendStatusFeedContext.Provider>
  );
}

export function useFriendStatusFeed() {
  const context = useContext(FriendStatusFeedContext);
  if (!context) {
    throw new Error(
      'useFriendStatusFeed must be used inside FriendStatusFeedProvider.',
    );
  }
  return context;
}
