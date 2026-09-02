import { Pressable, StyleSheet, Text } from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { ProfileForm } from '@/components/profile/ProfileForm';
import { SectionHeader } from '@/components/SectionHeader';
import { useAuth } from '@/providers/AuthProvider';
import { useProfile } from '@/providers/ProfileProvider';
import { tokens } from '@/theme/tokens';

export default function OnboardingScreen() {
  const { logout } = useAuth();
  const { profile, error, isLoading, refresh, update } = useProfile();

  return (
    <AppScreen>
      <SectionHeader
        title="Welcome to Dollhouse"
        subtitle="Choose how your friends will know and find you."
      />
      <AppCard>
        <Text style={styles.privacy}>
          Your sign-in email stays private. Your username and display name are
          the public identity your friends will see.
        </Text>
        {error ? (
          <Text accessibilityLiveRegion="assertive" style={styles.error}>
            {error} You can retry loading or continue by saving your identity.
          </Text>
        ) : null}
        <ProfileForm
          initialProfile={profile}
          key={profile?.updatedAt ?? 'new-profile'}
          onSave={update}
          submitLabel="Enter Dollhouse"
        />
      </AppCard>
      {error ? (
        <Pressable
          accessibilityRole="button"
          disabled={isLoading}
          onPress={() => void refresh()}
          style={styles.secondaryButton}
        >
          <Text style={styles.secondaryText}>
            {isLoading ? 'Retrying…' : 'Retry profile load'}
          </Text>
        </Pressable>
      ) : null}
      <Pressable
        accessibilityRole="button"
        onPress={() => void logout()}
        style={styles.signOut}
      >
        <Text style={styles.signOutText}>Sign out</Text>
      </Pressable>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  privacy: {
    marginBottom: tokens.spacing.lg,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 16,
    lineHeight: 23,
  },
  error: {
    marginBottom: tokens.spacing.md,
    padding: tokens.spacing.md,
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.bodySemibold,
    borderWidth: 1,
    borderColor: tokens.color.secondaryAccent,
    borderRadius: tokens.radius.sm,
  },
  secondaryButton: {
    minHeight: 48,
    marginTop: tokens.spacing.md,
    alignItems: 'center',
    justifyContent: 'center',
  },
  secondaryText: {
    color: tokens.color.accent,
    fontFamily: tokens.typography.headingSemibold,
  },
  signOut: {
    minHeight: 48,
    alignItems: 'center',
    justifyContent: 'center',
  },
  signOutText: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.headingSemibold,
  },
});
