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
    backgroundColor: '#F5DDD6',
    color: '#7D2C1D',
    fontSize: 14,
    lineHeight: 20,
  },
});
