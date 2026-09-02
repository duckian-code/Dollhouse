import { Link } from 'expo-router';
import { Pressable, StyleSheet, Text, View } from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { ProfileForm } from '@/components/profile/ProfileForm';
import { SectionHeader } from '@/components/SectionHeader';
import { useAuth } from '@/providers/AuthProvider';
import { useProfile } from '@/providers/ProfileProvider';
import { tokens } from '@/theme/tokens';

export default function ProfileScreen() {
  const { logout } = useAuth();
  const { profile, update } = useProfile();

  return (
    <AppScreen>
      <SectionHeader
        title="Profile"
        subtitle="Your avatar, preferences, and account."
      />
      <AppCard style={styles.profile}>
        <View
          accessible
          accessibilityLabel="Avatar placeholder"
          style={styles.avatar}
        >
          <Text style={styles.avatarText}>○</Text>
        </View>
        <Text style={styles.name}>{profile?.displayName}</Text>
        <Text style={styles.username}>@{profile?.username}</Text>
        <Text style={styles.detail}>
          {profile?.bio ?? 'Make your Dollhouse resident your own.'}
        </Text>
      </AppCard>
      <AppCard style={styles.editor}>
        <Text style={styles.editorTitle}>Edit profile</Text>
        <ProfileForm
          includeBio
          initialProfile={profile}
          onSave={update}
          submitLabel="Save profile"
        />
      </AppCard>
      <Link href="/avatar-customization" asChild>
        <Pressable
          accessibilityRole="link"
          style={({ pressed }) => [styles.customize, pressed && styles.pressed]}
        >
          <Text style={styles.customizeText}>Customize doll</Text>
        </Pressable>
      </Link>
      <Pressable
        accessibilityRole="button"
        onPress={logout}
        style={({ pressed }) => [styles.signOut, pressed && styles.pressed]}
      >
        <Text style={styles.signOutText}>Sign out</Text>
      </Pressable>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  profile: { alignItems: 'center', paddingVertical: tokens.spacing.xl },
  avatar: {
    width: 88,
    height: 88,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accentSoft,
  },
  avatarText: {
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingRegular,
    fontSize: 54,
  },
  name: {
    marginTop: tokens.spacing.md,
    color: tokens.color.text,
    fontSize: 19,
    fontFamily: tokens.typography.headingBold,
  },
  detail: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 16,
    textAlign: 'center',
  },
  username: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 15,
  },
  editor: { marginTop: tokens.spacing.lg },
  editorTitle: {
    marginBottom: tokens.spacing.md,
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 19,
  },
  signOut: {
    minHeight: 52,
    marginTop: tokens.spacing.lg,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    borderColor: tokens.color.accent,
    borderRadius: tokens.radius.round,
  },
  customize: {
    minHeight: 52,
    marginTop: tokens.spacing.lg,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  customizeText: {
    color: tokens.color.onAccent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 16,
  },
  pressed: { opacity: 0.7 },
  signOutText: {
    color: tokens.color.accent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 16,
  },
});
