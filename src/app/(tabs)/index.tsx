import { Link } from 'expo-router';
import { Pressable, StyleSheet, Text, View } from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { SectionHeader } from '@/components/SectionHeader';
import { tokens } from '@/theme/tokens';

export default function HomeScreen() {
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
          <Text style={styles.cardTitle}>Ready to check in</Text>
          <Text style={styles.cardBody}>
            Track how you feel and notice patterns over time.
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
  houseIcon: { color: tokens.color.accent, fontSize: 76, lineHeight: 82 },
  houseLabel: {
    color: tokens.color.textMuted,
    fontSize: 14,
    fontWeight: '600',
  },
  cards: { gap: tokens.spacing.md, marginTop: tokens.spacing.md },
  summaryCard: { minHeight: 126 },
  eyebrow: {
    color: tokens.color.textMuted,
    fontSize: 12,
    fontWeight: '700',
    letterSpacing: 0.8,
  },
  cardTitle: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.text,
    fontSize: 18,
    fontWeight: '700',
  },
  cardBody: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontSize: 14,
    lineHeight: 20,
  },
  button: {
    minHeight: 52,
    marginTop: tokens.spacing.lg,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accentSoft,
  },
  buttonPressed: { opacity: 0.82 },
  buttonText: { color: tokens.color.accent, fontSize: 16, fontWeight: '700' },
});
