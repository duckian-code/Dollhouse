import { ActivityIndicator, Pressable, StyleSheet, Text } from 'react-native';

import { tokens } from '@/theme/tokens';

type AuthButtonProps = {
  label: string;
  loading?: boolean;
  onPress: () => void;
};

export function AuthButton({
  label,
  loading = false,
  onPress,
}: AuthButtonProps) {
  return (
    <Pressable
      accessibilityRole="button"
      disabled={loading}
      onPress={onPress}
      style={({ pressed }) => [
        styles.button,
        (pressed || loading) && styles.buttonMuted,
      ]}
    >
      {loading ? (
        <ActivityIndicator color={tokens.color.white} />
      ) : (
        <Text style={styles.label}>{label}</Text>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  button: {
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  buttonMuted: { opacity: 0.75 },
  label: { color: tokens.color.white, fontSize: 16, fontWeight: '700' },
});
