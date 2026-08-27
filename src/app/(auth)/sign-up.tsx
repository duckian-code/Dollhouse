import { Link, router } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, Text } from 'react-native';

import { AuthButton } from '@/components/auth/AuthButton';
import { AuthMessage } from '@/components/auth/AuthMessage';
import { AuthShell } from '@/components/auth/AuthShell';
import { AuthTextField } from '@/components/auth/AuthTextField';
import { getAuthErrorMessage } from '@/services/auth/errors';
import { register } from '@/services/auth/cognito';
import { tokens } from '@/theme/tokens';

export default function SignUpScreen() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSignUp() {
    if (loading) return;
    if (!email.trim() || !password || !confirmPassword) {
      setError('Complete all fields to create your account.');
      return;
    }
    if (password !== confirmPassword) {
      setError('The passwords do not match.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const result = await register(email, password);
      if (result.nextStep.signUpStep === 'DONE') {
        router.replace({ pathname: '/', params: { email } });
      } else {
        router.replace({ pathname: '/confirm-sign-up', params: { email } });
      }
    } catch (caught) {
      setError(getAuthErrorMessage(caught));
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthShell
      title="Create your account"
      subtitle="Start building a space for you and your friends."
    >
      <AuthMessage>{error}</AuthMessage>
      <AuthTextField
        autoCapitalize="none"
        autoComplete="email"
        inputMode="email"
        label="Email"
        onChangeText={setEmail}
        placeholder="you@example.com"
        value={email}
      />
      <AuthTextField
        autoComplete="new-password"
        label="Password"
        onChangeText={setPassword}
        placeholder="At least 12 characters"
        secureTextEntry
        value={password}
      />
      <Text style={styles.hint}>
        Use uppercase, lowercase, a number, and a symbol.
      </Text>
      <AuthTextField
        autoComplete="new-password"
        label="Confirm password"
        onChangeText={setConfirmPassword}
        onSubmitEditing={handleSignUp}
        placeholder="Enter it again"
        returnKeyType="done"
        secureTextEntry
        value={confirmPassword}
      />
      <AuthButton
        label="Create account"
        loading={loading}
        onPress={handleSignUp}
      />
      <Link href="/" asChild>
        <Pressable accessibilityRole="link" style={styles.link}>
          <Text style={styles.linkText}>Already have an account? Sign in</Text>
        </Pressable>
      </Link>
    </AuthShell>
  );
}

const styles = StyleSheet.create({
  hint: {
    marginTop: -8,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 15,
  },
  link: { minHeight: 44, alignItems: 'center', justifyContent: 'center' },
  linkText: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 15,
  },
});
