import { Link } from 'expo-router';
import { Pressable, StyleSheet, Text, View } from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { SectionHeader } from '@/components/SectionHeader';
import { useMoodStatus } from '@/providers/MoodStatusProvider';
import { tokens } from '@/theme/tokens';

export default function HomeScreen() {
  const { currentStatus } = useMoodStatus();
  return (
    <AppScreen>
      <SectionHeader
        title="Good morning"
        subtitle="Welcome home to your Dollhouse."
      />
      <View accessible accessibilityLabel="Your Dollhouse" style={styles.house}>
        <Text style={styles.houseIcon}>⌂</Text>
        <Text style={styles.houseLabel}>Your Dollhouse</Text>
      </View>
      <View style={styles.cards}>
        <AppCard style={styles.summaryCard}>
          <Text style={styles.eyebrow}>RECENT MOOD</Text>
          <Text style={styles.cardTitle}>
            {currentStatus?.status ?? 'Ready to check in'}
          </Text>
          <Text style={styles.cardBody}>
            {currentStatus
              ? `Updated ${new Date(currentStatus.updatedAt).toLocaleTimeString(
                  [],
                  {
                    hour: 'numeric',
                    minute: '2-digit',
                  },
                )}`
              : 'Track how you feel and notice patterns over time.'}
          </Text>
        </AppCard>
        <AppCard style={styles.summaryCard}>
          <Text style={styles.eyebrow}>FRIENDS’ HOME</Text>
          <Text style={styles.cardTitle}>Your people live here</Text>
          <Text style={styles.cardBody}>
            Friend activity will appear in a future update.
          </Text>
        </AppCard>
      </View>
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
  cards: { gap: tokens.spacing.md, marginTop: tokens.spacing.md },
  summaryCard: { minHeight: 126 },
  eyebrow: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 12,
    letterSpacing: 0.8,
  },
  cardTitle: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.text,
    fontSize: 18,
    fontFamily: tokens.typography.headingBold,
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
});
