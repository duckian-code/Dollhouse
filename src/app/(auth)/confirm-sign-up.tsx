import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, Text } from 'react-native';

import { AuthButton } from '@/components/auth/AuthButton';
import { AuthMessage } from '@/components/auth/AuthMessage';
import { AuthShell } from '@/components/auth/AuthShell';
import { AuthTextField } from '@/components/auth/AuthTextField';
import {
  confirmRegistration,
  resendRegistrationCode,
} from '@/services/auth/cognito';
import { getAuthErrorMessage } from '@/services/auth/errors';
import { tokens } from '@/theme/tokens';

export default function ConfirmSignUpScreen() {
  const { email = '' } = useLocalSearchParams<{ email?: string }>();
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleConfirm() {
    if (!code.trim()) {
      setError('Enter the confirmation code from your email.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      await confirmRegistration(email, code);
      router.replace({ pathname: '/', params: { email } });
    } catch (caught) {
      setError(getAuthErrorMessage(caught));
    } finally {
      setLoading(false);
    }
  }

  async function handleResend() {
    setError('');
    setNotice('');
    try {
      await resendRegistrationCode(email);
      setNotice('A new code was sent to your email.');
    } catch (caught) {
      setError(getAuthErrorMessage(caught));
    }
  }

  return (
    <AuthShell
      title="Check your email"
      subtitle={`Enter the confirmation code sent to ${email || 'your email'}.`}
    >
      <AuthMessage>{error}</AuthMessage>
      {notice ? (
        <Text accessibilityLiveRegion="polite" style={styles.notice}>
          {notice}
        </Text>
      ) : null}
      <AuthTextField
        autoComplete="one-time-code"
        inputMode="numeric"
        label="Confirmation code"
        maxLength={6}
        onChangeText={setCode}
        onSubmitEditing={handleConfirm}
        placeholder="123456"
        returnKeyType="done"
        value={code}
      />
      <AuthButton
        label="Confirm email"
        loading={loading}
        onPress={handleConfirm}
      />
      <Pressable
        accessibilityRole="button"
        onPress={handleResend}
        style={styles.link}
      >
        <Text style={styles.linkText}>Send a new code</Text>
      </Pressable>
    </AuthShell>
  );
}

const styles = StyleSheet.create({
  notice: {
    padding: tokens.spacing.md,
    borderRadius: tokens.radius.sm,
    backgroundColor: tokens.color.accentSoft,
    color: tokens.color.text,
    fontSize: 14,
  },
  link: { minHeight: 44, alignItems: 'center', justifyContent: 'center' },
  linkText: { color: tokens.color.accent, fontSize: 15, fontWeight: '700' },
});
