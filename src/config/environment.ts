const DEFAULT_API_URL = 'http://localhost:3000';

export const environment = {
  apiUrl: process.env.EXPO_PUBLIC_API_URL ?? DEFAULT_API_URL,
  cognitoUserPoolId: process.env.EXPO_PUBLIC_COGNITO_USER_POOL_ID,
  cognitoUserPoolClientId: process.env.EXPO_PUBLIC_COGNITO_USER_POOL_CLIENT_ID,
} as const;

export const isAuthConfigured = Boolean(
  environment.cognitoUserPoolId && environment.cognitoUserPoolClientId,
);
