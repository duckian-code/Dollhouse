import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';

import { tokens } from '@/theme/tokens';

export function LoadingState({ label }: { label: string }) {
  return (
    <View accessibilityLabel={label} style={styles.state}>
      <ActivityIndicator color={tokens.color.accent} size="large" />
      <Text style={styles.detail}>{label}</Text>
    </View>
  );
}

export function EmptyStateMessage({
  title,
  detail,
}: {
  title: string;
  detail: string;
}) {
  return (
    <View style={styles.state}>
      <Text style={styles.title}>{title}</Text>
      <Text style={styles.detail}>{detail}</Text>
    </View>
  );
}

export function RetryState({
  message,
  retrying = false,
  onRetry,
}: {
  message: string;
  retrying?: boolean;
  onRetry: () => void;
}) {
  return (
    <View accessibilityLiveRegion="assertive" style={styles.error}>
      <Text style={styles.errorText}>{message}</Text>
      <Pressable
        accessibilityRole="button"
        accessibilityState={{ busy: retrying, disabled: retrying }}
        disabled={retrying}
        onPress={onRetry}
        style={styles.retryButton}
      >
        {retrying ? (
          <ActivityIndicator color={tokens.color.onAccent} />
        ) : (
          <Text style={styles.retryText}>Try again</Text>
        )}
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  state: { minHeight: 140, alignItems: 'center', justifyContent: 'center' },
  title: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 19,
    textAlign: 'center',
  },
  detail: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 16,
    textAlign: 'center',
  },
  error: {
    padding: tokens.spacing.md,
    borderWidth: 1,
    borderColor: tokens.color.secondaryAccent,
    borderRadius: tokens.radius.sm,
    backgroundColor: tokens.color.surface,
  },
  errorText: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 16,
  },
  retryButton: {
    minHeight: 42,
    marginTop: tokens.spacing.md,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  retryText: {
    color: tokens.color.onAccent,
    fontFamily: tokens.typography.headingBold,
  },
});
