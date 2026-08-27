import { Link } from 'expo-router';
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
import { FriendStatusCard } from '@/components/friends/FriendStatusCard';
import { SectionHeader } from '@/components/SectionHeader';
import { useFriendStatusFeed } from '@/providers/FriendStatusFeedProvider';
import { useMoodStatus } from '@/providers/MoodStatusProvider';
import { tokens } from '@/theme/tokens';

export default function HomeScreen() {
  const { currentStatus } = useMoodStatus();
  const {
    statuses,
    catalog,
    isInitialLoading,
    isRefreshing,
    error,
    notice,
    isPreview,
    refresh,
  } = useFriendStatusFeed();

  return (
    <AppScreen
      refreshControl={
        <RefreshControl
          colors={[tokens.color.accent]}
          onRefresh={() => void refresh()}
          refreshing={isRefreshing}
          tintColor={tokens.color.accent}
        />
      }
    >
      <SectionHeader
        title="Good morning"
        subtitle="Welcome home to your Dollhouse."
      />
      <View accessible accessibilityLabel="Your Dollhouse" style={styles.house}>
        <Text style={styles.houseIcon}>⌂</Text>
        <Text style={styles.houseLabel}>Your Dollhouse</Text>
      </View>

      <AppCard style={styles.moodCard}>
        <Text style={styles.eyebrow}>RECENT MOOD</Text>
        <Text style={styles.cardTitle}>
          {currentStatus?.status ?? 'Ready to check in'}
        </Text>
        <Text style={styles.cardBody}>
          {currentStatus
            ? `Updated ${new Date(currentStatus.updatedAt).toLocaleTimeString(
                [],
                { hour: 'numeric', minute: '2-digit' },
              )}`
            : 'Track how you feel and notice patterns over time.'}
        </Text>
      </AppCard>

      <Link href="/check-in" asChild>
        <Pressable
          accessibilityHint="Opens the mood check-in screen"
          accessibilityRole="button"
          style={({ pressed }) => [
            styles.button,
            pressed && styles.buttonPressed,
          ]}
        >
          <Text style={styles.buttonText}>＋ Check in</Text>
        </Pressable>
      </Link>

      <View style={styles.feedHeader}>
        <View style={styles.feedTitleGroup}>
          <Text style={styles.feedTitle}>Friends at home</Text>
          <Text style={styles.feedSubtitle}>
            See how your accepted friends are feeling.
          </Text>
        </View>
        {isRefreshing ? (
          <ActivityIndicator color={tokens.color.accent} />
        ) : null}
      </View>

      {isPreview ? (
        <Text style={styles.previewNotice}>
          Local preview — pull down to reset the sample status feed.
        </Text>
      ) : null}
      {notice ? <Text style={styles.notice}>{notice}</Text> : null}
      {error ? (
        <RetryState
          message={error}
          onRetry={() => void refresh()}
          retrying={isRefreshing}
        />
      ) : null}

      {isInitialLoading && !statuses.length ? (
        <LoadingState label="Opening your friends’ rooms…" />
      ) : statuses.length ? (
        <View style={styles.feed}>
          {statuses.map((item) => (
            <FriendStatusCard
              catalog={catalog}
              item={item}
              key={item.friend.userId}
            />
          ))}
        </View>
      ) : (
        <AppCard style={styles.emptyState}>
          <EmptyStateMessage
            detail="Accepted friends and their latest shared moods will appear here."
            title="No friends at home yet"
          />
        </AppCard>
      )}
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  house: {
    minHeight: 190,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.lg,
    backgroundColor: tokens.color.accentSoft,
  },
  houseIcon: {
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingRegular,
    fontSize: 76,
    lineHeight: 82,
  },
  houseLabel: {
    color: tokens.color.accent,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 14,
  },
  moodCard: { minHeight: 126, marginTop: tokens.spacing.md },
  eyebrow: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 12,
    letterSpacing: 0.8,
  },
  cardTitle: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 18,
  },
  cardBody: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 16,
    lineHeight: 22,
  },
  button: {
    minHeight: 52,
    marginTop: tokens.spacing.lg,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  buttonPressed: { opacity: 0.82 },
  buttonText: {
    color: tokens.color.onAccent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 16,
  },
  feedHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: tokens.spacing.xxl,
  },
  feedTitleGroup: { flex: 1 },
  feedTitle: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 25,
  },
  feedSubtitle: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 16,
  },
  previewNotice: {
    marginTop: tokens.spacing.md,
    padding: tokens.spacing.sm,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 15,
    borderRadius: tokens.radius.sm,
    backgroundColor: tokens.color.surfaceMuted,
  },
  notice: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 15,
  },
  feed: { gap: tokens.spacing.md, marginTop: tokens.spacing.md },
  emptyState: { marginTop: tokens.spacing.md, alignItems: 'center' },
});
