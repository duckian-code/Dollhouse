import 'react-native-get-random-values';
import 'react-native-url-polyfill/auto';

import { Amplify } from 'aws-amplify';
import {
  confirmSignUp,
  fetchAuthSession,
  getCurrentUser,
  resendSignUpCode,
  signIn,
  signOut,
  signUp,
} from 'aws-amplify/auth';
import { cognitoUserPoolsTokenProvider } from 'aws-amplify/auth/cognito';

import { environment, isAuthConfigured } from '@/config/environment';
import { authStorage } from '@/services/auth/secureStorage';

if (isAuthConfigured) {
  Amplify.configure({
    Auth: {
      Cognito: {
        userPoolId: environment.cognitoUserPoolId!,
        userPoolClientId: environment.cognitoUserPoolClientId!,
        loginWith: { email: true },
        signUpVerificationMethod: 'code',
        userAttributes: { email: { required: true } },
        passwordFormat: {
          minLength: 12,
          requireLowercase: true,
          requireUppercase: true,
          requireNumbers: true,
          requireSpecialCharacters: true,
        },
      },
    },
  });
  cognitoUserPoolsTokenProvider.setKeyValueStorage(authStorage);
}

function requireConfiguration() {
  if (!isAuthConfigured) {
    throw new Error(
      'Cognito is not configured. Add the user pool and client IDs to your Expo environment.',
    );
  }
}

export async function register(email: string, password: string) {
  requireConfiguration();
  return signUp({
    username: email.trim().toLowerCase(),
    password,
    options: { userAttributes: { email: email.trim().toLowerCase() } },
  });
}

export async function confirmRegistration(email: string, code: string) {
  requireConfiguration();
  return confirmSignUp({
    username: email.trim().toLowerCase(),
    confirmationCode: code.trim(),
  });
}

export async function resendRegistrationCode(email: string) {
  requireConfiguration();
  return resendSignUpCode({ username: email.trim().toLowerCase() });
}

export async function login(email: string, password: string) {
  requireConfiguration();
  return signIn({
    username: email.trim().toLowerCase(),
    password,
    options: { authFlowType: 'USER_SRP_AUTH' },
  });
}

export async function logout() {
  requireConfiguration();
  await signOut();
}

export async function hasAuthenticatedUser() {
  if (!isAuthConfigured) return false;
  try {
    await getCurrentUser();
    return true;
  } catch {
    return false;
  }
}

export async function getAccessToken() {
  if (!isAuthConfigured) return undefined;
  const session = await fetchAuthSession();
  return session.tokens?.accessToken?.toString();
}
