declare namespace NodeJS {
  interface ProcessEnv {
    readonly EXPO_PUBLIC_API_URL?: string;
    readonly EXPO_PUBLIC_COGNITO_USER_POOL_ID?: string;
    readonly EXPO_PUBLIC_COGNITO_USER_POOL_CLIENT_ID?: string;
  }
}
