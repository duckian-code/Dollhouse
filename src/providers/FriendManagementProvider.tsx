import {
  createContext,
  type PropsWithChildren,
  useCallback,
  useContext,
  useMemo,
  useState,
} from 'react';

import { useAuth } from '@/providers/AuthProvider';
import { listFriendRequests, searchUsers } from '@/services/api/dollhouse';
import {
  acceptFriendRequestAndRefresh,
  declineFriendRequestAndRefresh,
  loadFriendSummaries,
  removeFriendAndRefresh,
  sendFriendRequestAndRefresh,
} from '@/services/cache/dollhouse';
import type { FriendRequest, UserSummary } from '@/types/api';
import { getUserFacingError } from '@/services/resilience/errors';

const previewUsers: UserSummary[] = [
  { userId: 'preview-alex', username: 'alexj', displayName: 'Alex Johnson' },
  { userId: 'preview-riley', username: 'rileyk', displayName: 'Riley Kim' },
  { userId: 'preview-sam', username: 'samlee', displayName: 'Sam Lee' },
];

const previewIncoming: FriendRequest[] = [
  {
    requestId: 'preview-request-riley',
    user: previewUsers[1],
    status: 'PENDING_INCOMING',
    requestedAt: '2026-08-25T18:00:00Z',
  },
];

interface FriendManagementContextValue {
  friends: UserSummary[];
  incoming: FriendRequest[];
  outgoing: FriendRequest[];
  searchResults: UserSummary[];
  isLoading: boolean;
  isSearching: boolean;
  pendingActionId: string | null;
  error: string | null;
  isPreview: boolean;
  refresh: () => Promise<void>;
  search: (query: string) => Promise<void>;
  sendRequest: (user: UserSummary) => Promise<void>;
  acceptRequest: (request: FriendRequest) => Promise<void>;
  declineRequest: (request: FriendRequest) => Promise<void>;
  removeFriend: (friend: UserSummary) => Promise<void>;
  clearError: () => void;
}

const FriendManagementContext =
  createContext<FriendManagementContextValue | null>(null);

function messageFrom(caught: unknown) {
  return getUserFacingError(caught);
}

export function FriendManagementProvider({ children }: PropsWithChildren) {
  const { isAuthenticated } = useAuth();
  const isPreview = __DEV__ && !isAuthenticated;
  const [friends, setFriends] = useState<UserSummary[]>(
    isPreview ? [previewUsers[0]] : [],
  );
  const [incoming, setIncoming] = useState<FriendRequest[]>(
    isPreview ? previewIncoming : [],
  );
  const [outgoing, setOutgoing] = useState<FriendRequest[]>([]);
  const [searchResults, setSearchResults] = useState<UserSummary[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isSearching, setIsSearching] = useState(false);
  const [pendingActionId, setPendingActionId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (isPreview) return;
    setIsLoading(true);
    setError(null);
    try {
      const [friendLoad, requests] = await Promise.all([
        loadFriendSummaries(),
        listFriendRequests(),
      ]);
      if (friendLoad.cached) setFriends(friendLoad.cached.data);
      const refreshed = await friendLoad.refresh;
      setFriends(refreshed.data);
      setIncoming(requests.data.incoming);
      setOutgoing(requests.data.outgoing);
    } catch (caught) {
      setError(messageFrom(caught));
    } finally {
      setIsLoading(false);
    }
  }, [isPreview]);

  const search = useCallback(
    async (query: string) => {
      const normalized = query.trim();
      if (!normalized) {
        setSearchResults([]);
        setError('Enter a username to search.');
        return;
      }
      setIsSearching(true);
      setError(null);
      try {
        if (isPreview) {
          const lower = normalized.toLowerCase();
          setSearchResults(
            previewUsers.filter(
              (user) =>
                user.username.toLowerCase().startsWith(lower) &&
                !friends.some((friend) => friend.userId === user.userId),
            ),
          );
        } else {
          const response = await searchUsers(normalized);
          setSearchResults(response.data.items);
        }
      } catch (caught) {
        setError(messageFrom(caught));
      } finally {
        setIsSearching(false);
      }
    },
    [friends, isPreview],
  );

  const sendRequest = useCallback(
    async (user: UserSummary) => {
      setPendingActionId(user.userId);
      setError(null);
      try {
        const request = isPreview
          ? {
              requestId: `preview-outgoing-${user.userId}`,
              user,
              status: 'PENDING_OUTGOING' as const,
              requestedAt: new Date().toISOString(),
            }
          : (await sendFriendRequestAndRefresh({ userId: user.userId })).data
              .friendRequest;
        setOutgoing((current) => [request, ...current]);
      } catch (caught) {
        setError(messageFrom(caught));
      } finally {
        setPendingActionId(null);
      }
    },
    [isPreview],
  );

  const acceptRequest = useCallback(
    async (request: FriendRequest) => {
      setPendingActionId(request.requestId);
      setError(null);
      try {
        if (!isPreview) await acceptFriendRequestAndRefresh(request.requestId);
        setIncoming((current) =>
          current.filter((item) => item.requestId !== request.requestId),
        );
        setFriends((current) => [request.user, ...current]);
        if (!isPreview) await refresh();
      } catch (caught) {
        setError(messageFrom(caught));
      } finally {
        setPendingActionId(null);
      }
    },
    [isPreview, refresh],
  );

  const declineRequest = useCallback(
    async (request: FriendRequest) => {
      setPendingActionId(request.requestId);
      setError(null);
      try {
        if (!isPreview) await declineFriendRequestAndRefresh(request.requestId);
        setIncoming((current) =>
          current.filter((item) => item.requestId !== request.requestId),
        );
        if (!isPreview) await refresh();
      } catch (caught) {
        setError(messageFrom(caught));
      } finally {
        setPendingActionId(null);
      }
    },
    [isPreview, refresh],
  );

  const removeFriend = useCallback(
    async (friend: UserSummary) => {
      setPendingActionId(friend.userId);
      setError(null);
      try {
        if (!isPreview) await removeFriendAndRefresh(friend.userId);
        setFriends((current) =>
          current.filter((item) => item.userId !== friend.userId),
        );
        if (!isPreview) await refresh();
      } catch (caught) {
        setError(messageFrom(caught));
      } finally {
        setPendingActionId(null);
      }
    },
    [isPreview, refresh],
  );

  const value = useMemo<FriendManagementContextValue>(
    () => ({
      friends,
      incoming,
      outgoing,
      searchResults,
      isLoading,
      isSearching,
      pendingActionId,
      error,
      isPreview,
      refresh,
      search,
      sendRequest,
      acceptRequest,
      declineRequest,
      removeFriend,
      clearError: () => setError(null),
    }),
    [
      friends,
      incoming,
      outgoing,
      searchResults,
      isLoading,
      isSearching,
      pendingActionId,
      error,
      isPreview,
      refresh,
      search,
      sendRequest,
      acceptRequest,
      declineRequest,
      removeFriend,
    ],
  );

  return (
    <FriendManagementContext.Provider value={value}>
      {children}
    </FriendManagementContext.Provider>
  );
}

export function useFriendManagement() {
  const context = useContext(FriendManagementContext);
  if (!context) {
    throw new Error(
      'useFriendManagement must be used inside FriendManagementProvider.',
    );
  }
  return context;
}
