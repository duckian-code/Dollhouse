import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';

import { getUsernameAvailability } from '@/services/api/dollhouse';
import { ApiError } from '@/services/api/client';
import { getUserFacingError } from '@/services/resilience/errors';
import { tokens } from '@/theme/tokens';
import type { Profile, UpdateProfileRequest } from '@/types/api';

type ProfileFormProps = {
  initialProfile?: Profile | null;
  includeBio?: boolean;
  submitLabel: string;
  onSave: (request: UpdateProfileRequest) => Promise<unknown>;
};

export function ProfileForm({
  initialProfile,
  includeBio = false,
  submitLabel,
  onSave,
}: ProfileFormProps) {
  const [username, setUsername] = useState(initialProfile?.username ?? '');
  const [displayName, setDisplayName] = useState(
    initialProfile?.displayName ?? '',
  );
  const [bio, setBio] = useState(initialProfile?.bio ?? '');
  const [availabilityResult, setAvailabilityResult] = useState<{
    username: string;
    available: boolean;
  } | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const trimmedUsername = username.trim();
  const isCurrentUsername =
    Boolean(initialProfile?.username) &&
    trimmedUsername.toLocaleLowerCase() ===
      initialProfile?.username.trim().toLocaleLowerCase();
  const availability =
    !trimmedUsername || trimmedUsername.length > 50 || isCurrentUsername
      ? 'idle'
      : availabilityResult?.username === trimmedUsername
        ? availabilityResult.available
          ? 'available'
          : 'unavailable'
        : 'checking';

  useEffect(() => {
    if (!trimmedUsername || trimmedUsername.length > 50 || isCurrentUsername) {
      return;
    }
    let active = true;
    const timeout = setTimeout(() => {
      void getUsernameAvailability(trimmedUsername)
        .then(({ data }) => {
          if (active) {
            setAvailabilityResult({
              username: trimmedUsername,
              available: data.available,
            });
          }
        })
        .catch(() => undefined);
    }, 350);
    return () => {
      active = false;
      clearTimeout(timeout);
    };
  }, [isCurrentUsername, trimmedUsername]);

  async function submit() {
    const trimmedDisplayName = displayName.trim();
    if (!trimmedUsername || !trimmedDisplayName) {
      setError('Username and display name are required.');
      return;
    }
    if (trimmedUsername.length > 50 || trimmedDisplayName.length > 50) {
      setError('Username and display name must be 50 characters or fewer.');
      return;
    }
    if (availability === 'unavailable') {
      setError('That username is already taken. Choose another one.');
      return;
    }
    setIsSaving(true);
    setError(null);
    setMessage(null);
    try {
      await onSave({
        username: trimmedUsername,
        displayName: trimmedDisplayName,
        ...(includeBio ? { bio: bio.trim() || null } : {}),
      });
      setMessage('Profile saved.');
    } catch (caught) {
      if (caught instanceof ApiError && caught.status === 409) {
        setAvailabilityResult({
          username: trimmedUsername,
          available: false,
        });
        setError('That username was just claimed. Choose another one.');
      } else {
        setError(
          getUserFacingError(caught, 'Your profile could not be saved.'),
        );
      }
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <View style={styles.form}>
      <Field
        autoCapitalize="none"
        autoCorrect={false}
        label="Username"
        maxLength={50}
        onChangeText={(value) => {
          setUsername(value);
          setError(null);
        }}
        placeholder="Your public handle"
        value={username}
      />
      <Text accessibilityLiveRegion="polite" style={styles.availability}>
        {availability === 'checking'
          ? 'Checking availability…'
          : availability === 'available'
            ? 'Username is available.'
            : availability === 'unavailable'
              ? 'Username is already taken.'
              : isCurrentUsername
                ? 'This is your current username.'
                : 'Used by friends to find you.'}
      </Text>
      <Field
        autoCapitalize="words"
        label="Display name"
        maxLength={50}
        onChangeText={(value) => {
          setDisplayName(value);
          setError(null);
        }}
        placeholder="What should friends call you?"
        value={displayName}
      />
      {includeBio ? (
        <Field
          label="Bio"
          maxLength={280}
          multiline
          onChangeText={(value) => {
            setBio(value);
            setError(null);
          }}
          placeholder="Tell your friends a little about yourself."
          style={styles.bio}
          textAlignVertical="top"
          value={bio}
        />
      ) : null}
      {error ? (
        <Text accessibilityLiveRegion="assertive" style={styles.error}>
          {error}
        </Text>
      ) : null}
      {message ? (
        <Text accessibilityLiveRegion="polite" style={styles.success}>
          {message}
        </Text>
      ) : null}
      <Pressable
        accessibilityRole="button"
        accessibilityState={{ busy: isSaving, disabled: isSaving }}
        disabled={isSaving}
        onPress={() => void submit()}
        style={({ pressed }) => [
          styles.button,
          (pressed || isSaving) && styles.buttonMuted,
        ]}
      >
        {isSaving ? (
          <ActivityIndicator color={tokens.color.onAccent} />
        ) : (
          <Text style={styles.buttonText}>{submitLabel}</Text>
        )}
      </Pressable>
    </View>
  );
}

type FieldProps = React.ComponentProps<typeof TextInput> & { label: string };

function Field({ label, style, ...props }: FieldProps) {
  return (
    <View style={styles.field}>
      <Text style={styles.label}>{label}</Text>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={tokens.color.textMuted}
        selectionColor={tokens.color.accent}
        style={[styles.input, style]}
        {...props}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: tokens.spacing.md },
  field: { gap: tokens.spacing.sm },
  label: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 15,
  },
  input: {
    minHeight: 52,
    paddingHorizontal: tokens.spacing.md,
    borderWidth: 1,
    borderColor: tokens.color.border,
    borderRadius: tokens.radius.md,
    backgroundColor: tokens.color.surface,
    color: tokens.color.text,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 17,
  },
  bio: { minHeight: 112, paddingTop: tokens.spacing.md },
  availability: {
    marginTop: -tokens.spacing.sm,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 14,
  },
  error: {
    padding: tokens.spacing.md,
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.bodySemibold,
    borderWidth: 1,
    borderColor: tokens.color.secondaryAccent,
    borderRadius: tokens.radius.sm,
  },
  success: {
    color: tokens.color.highlight,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 15,
  },
  button: {
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  buttonMuted: { opacity: 0.7 },
  buttonText: {
    color: tokens.color.onAccent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 16,
  },
});
