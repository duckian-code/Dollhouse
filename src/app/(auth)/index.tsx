import { Link, router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, Text } from 'react-native';

import { AuthButton } from '@/components/auth/AuthButton';
import { AuthMessage } from '@/components/auth/AuthMessage';
import { AuthShell } from '@/components/auth/AuthShell';
import { AuthTextField } from '@/components/auth/AuthTextField';
import { isAuthConfigured } from '@/config/environment';
import { useAuth } from '@/providers/AuthProvider';
import { getAuthErrorMessage } from '@/services/auth/errors';
import { tokens } from '@/theme/tokens';

export default function SignInScreen() {
  const params = useLocalSearchParams<{ email?: string }>();
  const { login } = useAuth();
  const [email, setEmail] = useState(params.email ?? '');
  const [password, setPassword] = useState('');
  const [error, setError] = useState(
    isAuthConfigured
      ? ''
      : 'Add the Cognito user pool and client IDs to your Expo environment before signing in.',
  );
  const [loading, setLoading] = useState(false);

  async function handleSignIn() {
    if (!email.trim() || !password) {
      setError('Enter your email and password.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      await login(email, password);
    } catch (caught) {
      const message = getAuthErrorMessage(caught);
      setError(message);
      if (
        caught instanceof Error &&
        caught.name === 'UserNotConfirmedException'
      ) {
        router.push({ pathname: '/confirm-sign-up', params: { email } });
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthShell title="Welcome home" subtitle="Sign in to visit your Dollhouse.">
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
        autoComplete="current-password"
        label="Password"
        onChangeText={setPassword}
        onSubmitEditing={handleSignIn}
        placeholder="Your password"
        returnKeyType="done"
        secureTextEntry
        value={password}
      />
      <AuthButton label="Sign in" loading={loading} onPress={handleSignIn} />
      <Link href="/sign-up" asChild>
        <Pressable accessibilityRole="link" style={styles.link}>
          <Text style={styles.linkText}>New here? Create an account</Text>
        </Pressable>
      </Link>
    </AuthShell>
  );
}

const styles = StyleSheet.create({
  link: { minHeight: 44, alignItems: 'center', justifyContent: 'center' },
  linkText: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 15,
  },
});
