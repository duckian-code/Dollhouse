import { StyleSheet, Text } from 'react-native';

import { tokens } from '@/theme/tokens';

export function AuthMessage({ children }: { children?: string }) {
  if (!children) return null;
  return (
    <Text accessibilityLiveRegion="polite" style={styles.message}>
      {children}
    </Text>
  );
}

const styles = StyleSheet.create({
  message: {
    padding: tokens.spacing.md,
    borderRadius: tokens.radius.sm,
    borderWidth: 1,
    borderColor: tokens.color.secondaryAccent,
    backgroundColor: tokens.color.surface,
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 16,
    lineHeight: 20,
  },
});
