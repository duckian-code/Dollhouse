import { CormorantGaramond_400Regular } from '@expo-google-fonts/cormorant-garamond/400Regular';
import { CormorantGaramond_600SemiBold } from '@expo-google-fonts/cormorant-garamond/600SemiBold';
import { CormorantGaramond_700Bold } from '@expo-google-fonts/cormorant-garamond/700Bold';
import { Fraunces_400Regular } from '@expo-google-fonts/fraunces/400Regular';
import { Fraunces_600SemiBold } from '@expo-google-fonts/fraunces/600SemiBold';
import { Fraunces_700Bold } from '@expo-google-fonts/fraunces/700Bold';
import { useFonts } from 'expo-font';
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { ActivityIndicator, StyleSheet, View } from 'react-native';

import { ApplicationBanner } from '@/components/ApplicationBanner';
import { AuthProvider, useAuth } from '@/providers/AuthProvider';
import { FriendManagementProvider } from '@/providers/FriendManagementProvider';
import { FriendStatusFeedProvider } from '@/providers/FriendStatusFeedProvider';
import { MoodStatusProvider } from '@/providers/MoodStatusProvider';
import { ProfileProvider, useProfile } from '@/providers/ProfileProvider';
import { ResilienceProvider } from '@/providers/ResilienceProvider';
import { tokens } from '@/theme/tokens';

export default function RootLayout() {
  const [fontsLoaded] = useFonts({
    Fraunces_400Regular,
    Fraunces_600SemiBold,
    Fraunces_700Bold,
    CormorantGaramond_400Regular,
    CormorantGaramond_600SemiBold,
    CormorantGaramond_700Bold,
  });

  if (!fontsLoaded) return <LoadingScreen label="Loading Dollhouse" />;

  return (
    <ResilienceProvider>
      <AuthProvider>
        <ProfileProvider>
          <ApplicationBanner />
          <MoodStatusProvider>
            <FriendManagementProvider>
              <FriendStatusFeedProvider>
                <RootNavigator />
              </FriendStatusFeedProvider>
            </FriendManagementProvider>
          </MoodStatusProvider>
        </ProfileProvider>
        <StatusBar style="light" />
      </AuthProvider>
    </ResilienceProvider>
  );
}

function RootNavigator() {
  const { isAuthenticated, isLoading } = useAuth();
  const {
    profile,
    isLoading: isProfileLoading,
    error: profileError,
  } = useProfile();

  if (isLoading || (isAuthenticated && isProfileLoading && !profileError)) {
    return <LoadingScreen label="Restoring your session" />;
  }

  const hasCompletedOnboarding = profile?.onboardingComplete === true;
  const canUseMainApp =
    (__DEV__ && !isAuthenticated) ||
    (isAuthenticated && hasCompletedOnboarding);

  return (
    <Stack
      screenOptions={{
        contentStyle: { backgroundColor: tokens.color.background },
        headerShown: false,
      }}
    >
      <Stack.Protected guard={!isAuthenticated}>
        <Stack.Screen name="(auth)" />
      </Stack.Protected>
      <Stack.Protected guard={isAuthenticated && !hasCompletedOnboarding}>
        <Stack.Screen name="onboarding" />
      </Stack.Protected>
      <Stack.Protected guard={canUseMainApp}>
        <Stack.Screen name="(tabs)" />
      </Stack.Protected>
      <Stack.Protected guard={canUseMainApp}>
        <Stack.Screen
          name="friend-search"
          options={{ animation: 'slide_from_right' }}
        />
        <Stack.Screen
          name="check-in"
          options={{
            animation: 'slide_from_bottom',
            presentation: 'modal',
          }}
        />
      </Stack.Protected>
      <Stack.Protected guard={canUseMainApp}>
        <Stack.Screen
          name="avatar-customization"
          options={{ animation: 'slide_from_right' }}
        />
      </Stack.Protected>
    </Stack>
  );
}

function LoadingScreen({ label }: { label: string }) {
  return (
    <View accessibilityLabel={label} style={styles.loading}>
      <ActivityIndicator color={tokens.color.accent} size="large" />
    </View>
  );
}

const styles = StyleSheet.create({
  loading: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: tokens.color.background,
  },
});
