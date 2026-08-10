import { router } from 'expo-router';
import { Pressable, StyleSheet, Text, View } from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { SectionHeader } from '@/components/SectionHeader';
import { tokens } from '@/theme/tokens';

export default function CheckInScreen() {
  return (
    <AppScreen>
      <View style={styles.toolbar}>
        <Pressable
          accessibilityRole="button"
          hitSlop={8}
          onPress={() => router.back()}
          style={styles.closeButton}
        >
          <Text style={styles.closeText}>Cancel</Text>
        </Pressable>
      </View>
      <SectionHeader
        title="How do you feel?"
        subtitle="Take a moment to check in with yourself."
      />
      <AppCard style={styles.placeholder}>
        <Text style={styles.icon}>＋</Text>
        <Text style={styles.title}>Mood controls are coming next</Text>
        <Text style={styles.body}>
          This modal is ready for the mood check-in experience.
        </Text>
      </AppCard>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  toolbar: { minHeight: 44, alignItems: 'flex-end', justifyContent: 'center' },
  closeButton: {
    minHeight: 44,
    justifyContent: 'center',
    paddingHorizontal: tokens.spacing.sm,
  },
  closeText: { color: tokens.color.accent, fontSize: 16, fontWeight: '700' },
  placeholder: { alignItems: 'center', paddingVertical: tokens.spacing.xxl },
  icon: { color: tokens.color.accent, fontSize: 48 },
  title: {
    marginTop: tokens.spacing.md,
    color: tokens.color.text,
    fontSize: 19,
    fontWeight: '700',
    textAlign: 'center',
  },
  body: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.textMuted,
    fontSize: 14,
    lineHeight: 21,
    textAlign: 'center',
  },
});
