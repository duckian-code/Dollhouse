import type { ReactNode } from 'react';
import { StyleSheet, Text, View } from 'react-native';

import { tokens } from '@/theme/tokens';
import type { UserSummary } from '@/types/api';

export function FriendRow({
  user,
  detail,
  actions,
}: {
  user: UserSummary;
  detail?: string;
  actions?: ReactNode;
}) {
  const initials = user.displayName
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0])
    .join('')
    .toUpperCase();
  return (
    <View
      accessibilityLabel={`${user.displayName}, @${user.username}`}
      style={styles.row}
    >
      <View style={styles.avatar}>
        <Text style={styles.initials}>{initials}</Text>
      </View>
      <View style={styles.identity}>
        <Text style={styles.name}>{user.displayName}</Text>
        <Text style={styles.detail}>{detail ?? `@${user.username}`}</Text>
      </View>
      {actions ? <View style={styles.actions}>{actions}</View> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    minHeight: 72,
    flexDirection: 'row',
    alignItems: 'center',
    gap: tokens.spacing.md,
    paddingVertical: tokens.spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: tokens.color.border,
  },
  avatar: {
    width: 46,
    height: 46,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.surfaceMuted,
  },
  initials: {
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingBold,
    fontSize: 15,
  },
  identity: { flex: 1 },
  name: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 16,
  },
  detail: {
    marginTop: 2,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 15,
  },
  actions: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: tokens.spacing.xs,
  },
});
