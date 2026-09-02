import { router } from 'expo-router';
import { useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Modal,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { MoodPicker } from '@/components/moods/MoodPicker';
import { StateSlider } from '@/components/moods/StateSlider';
import { SectionHeader } from '@/components/SectionHeader';
import { useAuth } from '@/providers/AuthProvider';
import { useMoodStatus } from '@/providers/MoodStatusProvider';
import { publishMood } from '@/services/api/dollhouse';
import { isCatalogMood, moodCatalog } from '@/services/moods/catalog';
import { getUserFacingError } from '@/services/resilience/errors';
import { tokens } from '@/theme/tokens';
import type { MoodState, PublishMoodRequest } from '@/types/api';

type StateKey = 'stress' | 'fatigue' | 'discomfort';
type SliderState = Record<StateKey, { disclosed: boolean; value: number }>;

const initialSliders: SliderState = {
  stress: { disclosed: false, value: 5 },
  fatigue: { disclosed: false, value: 5 },
  discomfort: { disclosed: false, value: 5 },
};

const stateLabels: Record<StateKey, string> = {
  stress: 'Stress',
  fatigue: 'Fatigue',
  discomfort: 'Discomfort',
};

export default function CheckInScreen() {
  const { isAuthenticated } = useAuth();
  const { refreshCurrentStatus } = useMoodStatus();
  const [selectedStatus, setSelectedStatus] = useState<string | null>(null);
  const [sliders, setSliders] = useState(initialSliders);
  const [showConfirmation, setShowConfirmation] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [publishedStatus, setPublishedStatus] = useState<MoodState | null>(
    null,
  );

  const request = useMemo<PublishMoodRequest | null>(() => {
    if (!isCatalogMood(selectedStatus)) return null;
    return {
      status: selectedStatus,
      stress: sliders.stress.disclosed ? sliders.stress.value : null,
      fatigue: sliders.fatigue.disclosed ? sliders.fatigue.value : null,
      discomfort: sliders.discomfort.disclosed
        ? sliders.discomfort.value
        : null,
    };
  }, [selectedStatus, sliders]);

  function updateSlider(key: StateKey, update: Partial<SliderState[StateKey]>) {
    setSliders((current) => ({
      ...current,
      [key]: { ...current[key], ...update },
    }));
  }

  function requestConfirmation() {
    if (!request || isSubmitting) {
      setError('Choose one mood before publishing your check-in.');
      return;
    }
    setError(null);
    setShowConfirmation(true);
  }

  async function submit() {
    if (!request || isSubmitting) return;
    setIsSubmitting(true);
    setError(null);
    try {
      const status =
        __DEV__ && !isAuthenticated
          ? {
              status: request.status,
              stress: request.stress ?? null,
              fatigue: request.fatigue ?? null,
              discomfort: request.discomfort ?? null,
              updatedAt: new Date().toISOString(),
            }
          : (await publishMood(request)).data.status;
      refreshCurrentStatus(status);
      setPublishedStatus(status);
      setShowConfirmation(false);
    } catch (caught) {
      setShowConfirmation(false);
      setError(
        getUserFacingError(
          caught,
          'Your check-in could not be published. Please try again.',
        ),
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  if (publishedStatus) {
    return (
      <AppScreen contentContainerStyle={styles.successScreen}>
        <AppCard accessibilityLiveRegion="polite" style={styles.successCard}>
          <Text style={styles.successSymbol}>✓</Text>
          <Text style={styles.successTitle}>Check-in published</Text>
          <Text style={styles.successMood}>{publishedStatus.status}</Text>
          <Text style={styles.successBody}>
            Your displayed current status has been refreshed.
          </Text>
          <Pressable
            accessibilityRole="button"
            onPress={() => router.dismissTo('/(tabs)')}
            style={styles.primaryButton}
          >
            <Text style={styles.primaryButtonText}>Done</Text>
          </Pressable>
        </AppCard>
      </AppScreen>
    );
  }

  return (
    <AppScreen>
      <View style={styles.toolbar}>
        <Pressable
          accessibilityRole="button"
          hitSlop={8}
          onPress={() => router.dismissTo('/(tabs)')}
          style={styles.closeButton}
        >
          <Text style={styles.closeText}>Cancel</Text>
        </Pressable>
      </View>
      <SectionHeader
        title="How do you feel?"
        subtitle="Take a moment to check in with yourself."
      />
      {__DEV__ && !isAuthenticated ? (
        <Text style={styles.previewNotice}>
          Local preview — publishing updates this session without contacting
          AWS.
        </Text>
      ) : null}
      {error ? (
        <Text accessibilityLiveRegion="assertive" style={styles.error}>
          {error}
        </Text>
      ) : null}
      <Text style={styles.sectionTitle}>Choose one mood</Text>
      <MoodPicker
        moods={moodCatalog}
        onSelect={(status) => {
          setSelectedStatus(status);
          setError(null);
        }}
        selectedStatus={selectedStatus}
      />
      <AppCard style={styles.statesCard}>
        <Text style={styles.sectionTitle}>Share more if you want</Text>
        <Text style={styles.sectionDescription}>
          Each value is private unless you turn on its Share switch.
        </Text>
        {(Object.keys(stateLabels) as StateKey[]).map((key) => (
          <StateSlider
            disclosed={sliders[key].disclosed}
            key={key}
            label={stateLabels[key]}
            onDisclosureChange={(disclosed) => updateSlider(key, { disclosed })}
            onValueChange={(value) => updateSlider(key, { value })}
            value={sliders[key].value}
          />
        ))}
      </AppCard>
      <Pressable
        accessibilityRole="button"
        accessibilityState={{ disabled: !request || isSubmitting }}
        disabled={!request || isSubmitting}
        onPress={requestConfirmation}
        style={({ pressed }) => [
          styles.primaryButton,
          (!request || isSubmitting) && styles.buttonDisabled,
          pressed && styles.buttonPressed,
        ]}
      >
        <Text style={styles.primaryButtonText}>Review check-in</Text>
      </Pressable>

      <Modal
        animationType="fade"
        onRequestClose={() => !isSubmitting && setShowConfirmation(false)}
        transparent
        visible={showConfirmation}
      >
        <View style={styles.modalBackdrop}>
          <View
            accessibilityViewIsModal
            accessibilityLabel="Confirm your check-in"
            style={styles.modalCard}
          >
            <Text style={styles.modalTitle}>Publish this check-in?</Text>
            <Text style={styles.confirmMood}>{request?.status}</Text>
            {(Object.keys(stateLabels) as StateKey[]).map((key) => (
              <Text key={key} style={styles.confirmDetail}>
                {stateLabels[key]}:{' '}
                {sliders[key].disclosed
                  ? `${sliders[key].value} out of 10`
                  : 'Undisclosed'}
              </Text>
            ))}
            {isSubmitting ? (
              <ActivityIndicator
                color={tokens.color.accent}
                style={styles.submitting}
              />
            ) : (
              <View style={styles.modalActions}>
                <Pressable
                  accessibilityRole="button"
                  onPress={() => setShowConfirmation(false)}
                  style={styles.secondaryButton}
                >
                  <Text style={styles.secondaryButtonText}>Keep editing</Text>
                </Pressable>
                <Pressable
                  accessibilityRole="button"
                  onPress={() => void submit()}
                  style={styles.primaryModalButton}
                >
                  <Text style={styles.primaryButtonText}>Publish</Text>
                </Pressable>
              </View>
            )}
          </View>
        </View>
      </Modal>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  toolbar: { minHeight: 44, alignItems: 'flex-end', justifyContent: 'center' },
  closeButton: {
    minHeight: 44,
    justifyContent: 'center',
    paddingHorizontal: tokens.spacing.sm,
  },
  closeText: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 16,
  },
  previewNotice: {
    marginBottom: tokens.spacing.lg,
    padding: tokens.spacing.sm,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 15,
    textAlign: 'center',
    borderRadius: tokens.radius.sm,
    backgroundColor: tokens.color.surfaceMuted,
  },
  error: {
    marginBottom: tokens.spacing.md,
    padding: tokens.spacing.md,
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 16,
    borderWidth: 1,
    borderColor: tokens.color.secondaryAccent,
    borderRadius: tokens.radius.sm,
  },
  sectionTitle: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 20,
  },
  sectionDescription: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 16,
  },
  statesCard: { marginTop: tokens.spacing.lg },
  primaryButton: {
    minHeight: 54,
    marginTop: tokens.spacing.lg,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: tokens.spacing.lg,
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  primaryButtonText: {
    color: tokens.color.onAccent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 16,
  },
  buttonDisabled: { opacity: 0.38 },
  buttonPressed: { opacity: 0.8 },
  modalBackdrop: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: tokens.spacing.lg,
    backgroundColor: 'rgba(0, 0, 0, 0.7)',
  },
  modalCard: {
    width: '100%',
    maxWidth: 480,
    padding: tokens.spacing.lg,
    borderWidth: 1,
    borderColor: tokens.color.border,
    borderRadius: tokens.radius.lg,
    backgroundColor: tokens.color.surface,
  },
  modalTitle: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 24,
  },
  confirmMood: {
    marginVertical: tokens.spacing.md,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingBold,
    fontSize: 30,
  },
  confirmDetail: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 17,
  },
  modalActions: {
    flexDirection: 'row',
    gap: tokens.spacing.sm,
    marginTop: tokens.spacing.lg,
  },
  secondaryButton: {
    flex: 1,
    minHeight: 50,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: tokens.spacing.md,
    borderWidth: 1,
    borderColor: tokens.color.border,
    borderRadius: tokens.radius.round,
  },
  secondaryButtonText: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 15,
  },
  primaryModalButton: {
    flex: 1,
    minHeight: 50,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: tokens.spacing.md,
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  submitting: { marginTop: tokens.spacing.lg },
  successScreen: { justifyContent: 'center' },
  successCard: { alignItems: 'center', padding: tokens.spacing.xl },
  successSymbol: {
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingBold,
    fontSize: 52,
  },
  successTitle: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 26,
  },
  successMood: {
    marginTop: tokens.spacing.md,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingBold,
    fontSize: 34,
  },
  successBody: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 17,
    textAlign: 'center',
  },
});
