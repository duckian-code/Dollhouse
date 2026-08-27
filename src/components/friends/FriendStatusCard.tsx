import { StyleSheet, Text, View } from 'react-native';

import { AppCard } from '@/components/AppCard';
import { FriendDollhouse } from '@/components/friends/FriendDollhouse';
import { moodCatalog } from '@/services/moods/catalog';
import { tokens } from '@/theme/tokens';
import type { AvatarCatalog, FriendStatus, MoodState } from '@/types/api';

interface FriendStatusCardProps {
  item: FriendStatus;
  catalog: AvatarCatalog | null;
}

function formatUpdatedAt(value: string) {
  const date = new Date(value);
  const elapsedMinutes = Math.floor((Date.now() - date.getTime()) / 60000);
  if (elapsedMinutes < 1) return 'Just now';
  if (elapsedMinutes < 60) return `${elapsedMinutes}m ago`;
  if (elapsedMinutes < 24 * 60)
    return `${Math.floor(elapsedMinutes / 60)}h ago`;
  return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
}

function DisclosedValues({ status }: { status: MoodState }) {
  const values = [
    ['Stress', status.stress],
    ['Fatigue', status.fatigue],
    ['Discomfort', status.discomfort],
  ].filter((entry): entry is [string, number] => entry[1] !== null);

  if (!values.length) {
    return <Text style={styles.privateCopy}>No state details shared.</Text>;
  }

  return (
    <View accessibilityLabel="Disclosed state values" style={styles.values}>
      {values.map(([label, value]) => (
        <View key={label} style={styles.valuePill}>
          <Text style={styles.valueText}>
            {label} · {value}
          </Text>
        </View>
      ))}
    </View>
  );
}

export function FriendStatusCard({ item, catalog }: FriendStatusCardProps) {
  const mood = item.status
    ? moodCatalog.find((entry) => entry.status === item.status?.status)
    : null;

  return (
    <AppCard style={styles.card}>
      <View style={styles.identityRow}>
        <View style={styles.initials}>
          <Text style={styles.initialsText}>
            {item.friend.displayName
              .split(/\s+/)
              .map((part) => part[0])
              .join('')
              .slice(0, 2)
              .toUpperCase()}
          </Text>
        </View>
        <View style={styles.identity}>
          <Text style={styles.name}>{item.friend.displayName}</Text>
          <Text style={styles.username}>@{item.friend.username}</Text>
        </View>
        {item.status ? (
          <Text
            accessibilityLabel={`Updated ${new Date(
              item.status.updatedAt,
            ).toLocaleString()}`}
            style={styles.timestamp}
          >
            {formatUpdatedAt(item.status.updatedAt)}
          </Text>
        ) : null}
      </View>

      {catalog ? (
        <FriendDollhouse
          catalog={catalog}
          doll={item.doll}
          friendName={item.friend.displayName}
        />
      ) : (
        <View style={styles.artLoading}>
          <Text style={styles.privateCopy}>Avatar art is loading…</Text>
        </View>
      )}

      {item.status ? (
        <>
          <View style={styles.moodRow}>
            <Text style={styles.moodLabel}>Current mood</Text>
            <Text style={styles.moodValue}>
              {item.status.status} {mood?.symbol ?? '◇'}
            </Text>
          </View>
          <DisclosedValues status={item.status} />
        </>
      ) : (
        <View style={styles.noStatus}>
          <Text style={styles.noStatusTitle}>No mood shared yet</Text>
          <Text style={styles.privateCopy}>
            Their room is here whenever they are ready to check in.
          </Text>
        </View>
      )}
    </AppCard>
  );
}

const styles = StyleSheet.create({
  card: { padding: tokens.spacing.md },
  identityRow: { flexDirection: 'row', alignItems: 'center' },
  initials: {
    width: 44,
    height: 44,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.surfaceMuted,
  },
  initialsText: {
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingBold,
    fontSize: 15,
  },
  identity: { flex: 1, marginLeft: tokens.spacing.sm },
  name: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 18,
  },
  username: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 15,
  },
  timestamp: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 14,
  },
  artLoading: {
    minHeight: 150,
    alignItems: 'center',
    justifyContent: 'center',
  },
  moodRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  moodLabel: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 16,
  },
  moodValue: {
    color: tokens.color.accent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 18,
  },
  values: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: tokens.spacing.sm,
    marginTop: tokens.spacing.md,
  },
  valuePill: {
    paddingHorizontal: tokens.spacing.sm,
    paddingVertical: tokens.spacing.xs,
    borderWidth: 1,
    borderColor: tokens.color.border,
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.surfaceMuted,
  },
  valueText: {
    color: tokens.color.text,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 14,
  },
  noStatus: { marginTop: tokens.spacing.sm },
  noStatusTitle: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 17,
  },
  privateCopy: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 15,
  },
});
