import type { PropsWithChildren } from 'react';
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { tokens } from '@/theme/tokens';

type AuthShellProps = PropsWithChildren<{
  title: string;
  subtitle: string;
}>;

export function AuthShell({ children, subtitle, title }: AuthShellProps) {
  return (
    <SafeAreaView style={styles.safeArea}>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        style={styles.fill}
      >
        <ScrollView
          contentContainerStyle={styles.content}
          keyboardShouldPersistTaps="handled"
        >
          <View accessible accessibilityLabel="Dollhouse" style={styles.logo}>
            <Text style={styles.logoIcon}>⌂</Text>
          </View>
          <Text accessibilityRole="header" style={styles.title}>
            {title}
          </Text>
          <Text style={styles.subtitle}>{subtitle}</Text>
          <View style={styles.form}>{children}</View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: { flex: 1, backgroundColor: tokens.color.background },
  fill: { flex: 1 },
  content: {
    flexGrow: 1,
    width: '100%',
    maxWidth: 520,
    alignSelf: 'center',
    justifyContent: 'center',
    padding: tokens.spacing.lg,
  },
  logo: {
    width: 72,
    height: 72,
    alignItems: 'center',
    justifyContent: 'center',
    alignSelf: 'center',
    borderRadius: tokens.radius.lg,
    backgroundColor: tokens.color.accentSoft,
  },
  logoIcon: {
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingRegular,
    fontSize: 48,
    lineHeight: 54,
  },
  title: {
    marginTop: tokens.spacing.lg,
    color: tokens.color.text,
    fontSize: tokens.typography.title,
    fontFamily: tokens.typography.headingBold,
    textAlign: 'center',
  },
  subtitle: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.textMuted,
    fontSize: tokens.typography.body,
    fontFamily: tokens.typography.bodyRegular,
    lineHeight: 24,
    textAlign: 'center',
  },
  form: { gap: tokens.spacing.md, marginTop: tokens.spacing.xl },
});
