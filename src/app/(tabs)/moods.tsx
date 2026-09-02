import { useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import {
  EmptyStateMessage,
  LoadingState,
  RetryState,
} from '@/components/AsyncState';
import { SectionHeader } from '@/components/SectionHeader';
import { useAuth } from '@/providers/AuthProvider';
import { getMoods } from '@/services/api/dollhouse';
import { getUserFacingError } from '@/services/resilience/errors';
import { tokens } from '@/theme/tokens';
import type { MoodEntry } from '@/types/api';

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  });
}

function sliderSummary(entry: MoodEntry) {
  const values = [
    entry.stress === null ? null : `Stress ${entry.stress}/10`,
    entry.fatigue === null ? null : `Fatigue ${entry.fatigue}/10`,
    entry.discomfort === null ? null : `Discomfort ${entry.discomfort}/10`,
  ].filter((value): value is string => value !== null);
  return values.join(' · ');
}

export default function MoodsScreen() {
  const { isAuthenticated } = useAuth();
  const [entries, setEntries] = useState<MoodEntry[]>([]);
  const [nextToken, setNextToken] = useState<string | null>(null);
  const [isInitialLoading, setIsInitialLoading] = useState(isAuthenticated);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadFirstPage = useCallback(async () => {
    if (!isAuthenticated) {
      setIsInitialLoading(false);
      return;
    }
    setIsRefreshing(true);
    setError(null);
    try {
      const response = await getMoods();
      setEntries(response.data.items);
      setNextToken(response.data.nextToken);
    } catch (caught) {
      setError(
        getUserFacingError(
          caught,
          'Your mood history could not be loaded. Please try again.',
        ),
      );
    } finally {
      setIsInitialLoading(false);
      setIsRefreshing(false);
    }
  }, [isAuthenticated]);

  useFocusEffect(
    useCallback(() => {
      void loadFirstPage();
    }, [loadFirstPage]),
  );

  const loadMore = useCallback(async () => {
    if (!nextToken || isLoadingMore) return;
    setIsLoadingMore(true);
    setError(null);
    try {
      const response = await getMoods(nextToken);
      setEntries((current) => {
        const existing = new Set(current.map((entry) => entry.eventId));
        return [
          ...current,
          ...response.data.items.filter(
            (entry) => !existing.has(entry.eventId),
          ),
        ];
      });
      setNextToken(response.data.nextToken);
    } catch (caught) {
      setError(
        getUserFacingError(
          caught,
          'Older mood entries could not be loaded. Please try again.',
        ),
      );
    } finally {
      setIsLoadingMore(false);
    }
  }, [isLoadingMore, nextToken]);

  return (
    <AppScreen
      refreshControl={
        isAuthenticated ? (
          <RefreshControl
            refreshing={isRefreshing && !isInitialLoading}
            tintColor={tokens.color.accent}
            onRefresh={() => void loadFirstPage()}
          />
        ) : undefined
      }
    >
      <SectionHeader
        title="Moods"
        subtitle="Your check-in history, newest first."
      />

      {isInitialLoading ? (
        <LoadingState label="Loading your mood history" />
      ) : error && entries.length === 0 ? (
        <RetryState
          message={error}
          retrying={isRefreshing}
          onRetry={() => void loadFirstPage()}
        />
      ) : entries.length === 0 ? (
        <EmptyStateMessage
          title="No check-ins yet"
          detail={
            isAuthenticated
              ? 'Your mood history will appear here after your first check-in.'
              : 'Sign in and complete a check-in to see your mood history.'
          }
        />
      ) : (
        <View style={styles.list}>
          {entries.map((entry) => {
            const details = sliderSummary(entry);
            return (
              <AppCard key={entry.eventId}>
                <Text style={styles.status}>{entry.status}</Text>
                <Text style={styles.date}>{formatDate(entry.updatedAt)}</Text>
                {details ? <Text style={styles.details}>{details}</Text> : null}
              </AppCard>
            );
          })}

          {error ? <Text style={styles.error}>{error}</Text> : null}
          {nextToken ? (
            <Pressable
              accessibilityRole="button"
              accessibilityState={{
                busy: isLoadingMore,
                disabled: isLoadingMore,
              }}
              disabled={isLoadingMore}
              onPress={() => void loadMore()}
              style={styles.loadMore}
            >
              {isLoadingMore ? (
                <ActivityIndicator color={tokens.color.onAccent} />
              ) : (
                <Text style={styles.loadMoreText}>Load older check-ins</Text>
              )}
            </Pressable>
          ) : null}
        </View>
      )}
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  list: { gap: tokens.spacing.md },
  status: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 19,
  },
  date: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 15,
  },
  details: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.text,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 16,
  },
  error: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 16,
    textAlign: 'center',
  },
  loadMore: {
    minHeight: 46,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  loadMoreText: {
    color: tokens.color.onAccent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 16,
  },
});
