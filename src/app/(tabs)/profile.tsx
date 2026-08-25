import { StyleSheet, Text, View } from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { SectionHeader } from '@/components/SectionHeader';
import { tokens } from '@/theme/tokens';

export default function ProfileScreen() {
  return (
    <AppScreen>
      <SectionHeader
        title="Profile"
        subtitle="Your avatar, preferences, and account."
      />
      <AppCard style={styles.profile}>
        <View
          accessible
          accessibilityLabel="Avatar placeholder"
          style={styles.avatar}
        >
          <Text style={styles.avatarText}>○</Text>
        </View>
        <Text style={styles.name}>Dollhouse resident</Text>
        <Text style={styles.detail}>Profile customization is coming soon.</Text>
      </AppCard>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  profile: { alignItems: 'center', paddingVertical: tokens.spacing.xl },
  avatar: {
    width: 88,
    height: 88,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accentSoft,
  },
  avatarText: { color: tokens.color.accent, fontSize: 54 },
  name: {
    marginTop: tokens.spacing.md,
    color: tokens.color.text,
    fontSize: 19,
    fontWeight: '700',
  },
  detail: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.textMuted,
    fontSize: 14,
    textAlign: 'center',
  },
});
