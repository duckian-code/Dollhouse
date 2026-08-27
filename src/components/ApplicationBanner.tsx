import { StyleSheet, Text, View } from 'react-native';

import { useResilience } from '@/providers/ResilienceProvider';
import { tokens } from '@/theme/tokens';

export function ApplicationBanner() {
  const { isOffline } = useResilience();
  if (!isOffline) return null;
  return (
    <View accessibilityLiveRegion="assertive" style={styles.banner}>
      <Text style={styles.text}>
        You’re offline. Saved information may still be available.
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  banner: {
    zIndex: 10,
    paddingHorizontal: tokens.spacing.md,
    paddingVertical: tokens.spacing.sm,
    backgroundColor: tokens.color.secondaryAccent,
  },
  text: {
    color: tokens.color.onAccent,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 15,
    textAlign: 'center',
  },
});
