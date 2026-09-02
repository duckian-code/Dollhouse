import { useFocusEffect } from 'expo-router';
import { useCallback, useMemo, useState } from 'react';
import { RefreshControl, StyleSheet, Text, View } from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { LoadingState, RetryState } from '@/components/AsyncState';
import { SectionHeader } from '@/components/SectionHeader';
import { useAuth } from '@/providers/AuthProvider';
import { useMoodStatus } from '@/providers/MoodStatusProvider';
import { getMoodEntries } from '@/services/api/dollhouse';
import { getUserFacingError } from '@/services/resilience/errors';
import { tokens } from '@/theme/tokens';
import type { MoodEntry } from '@/types/api';

const maxPages = 20;

function startOfToday(now = new Date()) {
  return new Date(now.getFullYear(), now.getMonth(), now.getDate());
}

function startOfWeek(now = new Date()) {
  const start = startOfToday(now);
  start.setDate(start.getDate() - start.getDay());
  return start;
}

export default function MoodsScreen() {
  const { isAuthenticated } = useAuth();
  const { currentStatus } = useMoodStatus();
  const [entries, setEntries] = useState<MoodEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (__DEV__ && !isAuthenticated) {
      setEntries(
        currentStatus ? [{ eventId: 'preview-current', ...currentStatus }] : [],
      );
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const allEntries: MoodEntry[] = [];
      const seenTokens = new Set<string>();
      const cutoff = startOfWeek().getTime();
      let nextToken: string | undefined;
      for (let page = 0; page < maxPages; page += 1) {
        const response = await getMoodEntries(nextToken);
        allEntries.push(...response.data.items);
        const oldest = response.data.items.at(-1);
        const reachedCutoff =
          oldest !== undefined && Date.parse(oldest.updatedAt) < cutoff;
        const followingToken = response.data.nextToken ?? undefined;
        if (
          reachedCutoff ||
          !followingToken ||
          seenTokens.has(followingToken)
        ) {
          break;
        }
        seenTokens.add(followingToken);
        nextToken = followingToken;
      }
      setEntries(allEntries);
    } catch (caught) {
      setError(
        getUserFacingError(caught, 'Your mood entries could not be loaded.'),
      );
    } finally {
      setIsLoading(false);
    }
  }, [currentStatus, isAuthenticated]);

  useFocusEffect(
    useCallback(() => {
      void refresh();
    }, [refresh]),
  );

  const { today, thisWeek } = useMemo(() => {
    const todayStart = startOfToday().getTime();
    const weekStart = startOfWeek().getTime();
    return {
      today: entries.filter(
        (entry) => Date.parse(entry.updatedAt) >= todayStart,
      ),
      thisWeek: entries.filter((entry) => {
        const timestamp = Date.parse(entry.updatedAt);
        return timestamp >= weekStart && timestamp < todayStart;
      }),
    };
  }, [entries]);

  return (
    <AppScreen
      refreshControl={
        <RefreshControl
          onRefresh={() => void refresh()}
          refreshing={isLoading}
          tintColor={tokens.color.accent}
        />
      }
    >
      <SectionHeader
        title="Moods"
        subtitle="Your check-in history, newest first."
      />
      {error ? (
        <RetryState
          message={error}
          onRetry={() => void refresh()}
          retrying={isLoading}
        />
      ) : null}
      {isLoading && entries.length === 0 ? (
        <LoadingState label="Opening your mood entries…" />
      ) : (
        <View style={styles.list}>
          <EntrySection
            emptyMessage="No check-ins yet today."
            entries={today}
            title="Today"
          />
          <EntrySection
            emptyMessage="No earlier check-ins this week."
            entries={thisWeek}
            title="This week"
          />
          <AppCard>
            <Text style={styles.title}>Patterns</Text>
            <Text style={styles.empty}>
              Insights are coming in a later update
            </Text>
          </AppCard>
        </View>
      )}
    </AppScreen>
  );
}

function EntrySection({
  title,
  entries,
  emptyMessage,
}: {
  title: string;
  entries: MoodEntry[];
  emptyMessage: string;
}) {
  return (
    <AppCard>
      <Text style={styles.title}>{title}</Text>
      {entries.length ? (
        <View style={styles.entries}>
          {entries.map((entry) => (
            <View key={entry.eventId} style={styles.entry}>
              <View style={styles.entryHeader}>
                <Text style={styles.status}>{entry.status}</Text>
                <Text style={styles.time}>
                  {new Date(entry.updatedAt).toLocaleString([], {
                    weekday: title === 'Today' ? undefined : 'short',
                    hour: 'numeric',
                    minute: '2-digit',
                  })}
                </Text>
              </View>
              <StateDetails entry={entry} />
            </View>
          ))}
        </View>
      ) : (
        <Text style={styles.empty}>{emptyMessage}</Text>
      )}
    </AppCard>
  );
}

function StateDetails({ entry }: { entry: MoodEntry }) {
  const details = [
    entry.stress === null ? null : `Stress ${entry.stress}/10`,
    entry.fatigue === null ? null : `Fatigue ${entry.fatigue}/10`,
    entry.discomfort === null ? null : `Discomfort ${entry.discomfort}/10`,
  ].filter((detail): detail is string => detail !== null);
  return details.length ? (
    <Text style={styles.details}>{details.join('  ·  ')}</Text>
  ) : null;
}

const styles = StyleSheet.create({
  list: { gap: tokens.spacing.md },
  title: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 19,
  },
  entries: { marginTop: tokens.spacing.sm },
  entry: {
    paddingVertical: tokens.spacing.md,
    borderTopWidth: 1,
    borderTopColor: tokens.color.border,
  },
  entryHeader: {
    flexDirection: 'row',
    alignItems: 'baseline',
    justifyContent: 'space-between',
    gap: tokens.spacing.md,
  },
  status: {
    flex: 1,
    color: tokens.color.text,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 17,
  },
  time: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 14,
  },
  details: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 14,
  },
  empty: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 16,
  },
});
