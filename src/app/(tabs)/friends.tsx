import { StyleSheet, Text } from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { SectionHeader } from '@/components/SectionHeader';
import { tokens } from '@/theme/tokens';

export default function FriendsScreen() {
  return (
    <AppScreen>
      <SectionHeader
        title="Friends"
        subtitle="Visit the people who share your Dollhouse."
      />
      <AppCard style={styles.emptyState}>
        <Text style={styles.icon}>♧</Text>
        <Text style={styles.title}>Make this house a home</Text>
        <Text style={styles.body}>
          Your friends and their avatars will appear here.
        </Text>
      </AppCard>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  emptyState: { alignItems: 'center', paddingVertical: tokens.spacing.xxl },
  icon: {
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingRegular,
    fontSize: 50,
  },
  title: {
    marginTop: tokens.spacing.md,
    color: tokens.color.text,
    fontSize: 19,
    fontFamily: tokens.typography.headingBold,
  },
  body: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 16,
    textAlign: 'center',
  },
});
