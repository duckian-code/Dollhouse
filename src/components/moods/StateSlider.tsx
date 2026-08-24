import Slider from '@react-native-community/slider';
import { StyleSheet, Switch, Text, View } from 'react-native';

import { tokens } from '@/theme/tokens';

interface StateSliderProps {
  label: string;
  value: number;
  disclosed: boolean;
  onDisclosureChange: (disclosed: boolean) => void;
  onValueChange: (value: number) => void;
}

export function StateSlider({
  label,
  value,
  disclosed,
  onDisclosureChange,
  onValueChange,
}: StateSliderProps) {
  const lowerLabel = label.toLowerCase();
  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <View>
          <Text style={styles.label}>{label}</Text>
          <Text style={styles.detail}>
            {disclosed ? `${value} out of 10` : 'Undisclosed'}
          </Text>
        </View>
        <View style={styles.toggleRow}>
          <Text style={styles.toggleLabel}>Share</Text>
          <Switch
            accessibilityLabel={`Share ${lowerLabel}`}
            onValueChange={onDisclosureChange}
            thumbColor={
              disclosed ? tokens.color.highlight : tokens.color.textMuted
            }
            trackColor={{
              false: tokens.color.surfaceMuted,
              true: tokens.color.accent,
            }}
            value={disclosed}
          />
        </View>
      </View>
      <Slider
        accessibilityLabel={`${label} level`}
        disabled={!disclosed}
        maximumTrackTintColor={tokens.color.muted}
        maximumValue={10}
        minimumTrackTintColor={tokens.color.highlight}
        minimumValue={0}
        onValueChange={onValueChange}
        step={1}
        style={[styles.slider, !disclosed && styles.sliderDisabled]}
        thumbTintColor={tokens.color.accent}
        value={value}
      />
      <View style={styles.bounds}>
        <Text style={styles.bound}>0 · Low</Text>
        <Text style={styles.bound}>10 · High</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingVertical: tokens.spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: tokens.color.border,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: tokens.spacing.md,
  },
  label: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 17,
  },
  detail: {
    marginTop: 2,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 15,
  },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: tokens.spacing.sm,
  },
  toggleLabel: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 15,
  },
  slider: { width: '100%', height: 38, marginTop: tokens.spacing.sm },
  sliderDisabled: { opacity: 0.3 },
  bounds: { flexDirection: 'row', justifyContent: 'space-between' },
  bound: {
    color: tokens.color.muted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 13,
  },
});
