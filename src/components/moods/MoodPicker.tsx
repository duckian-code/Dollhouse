import { Pressable, StyleSheet, Text, View } from 'react-native';

import type { MoodOption } from '@/services/moods/catalog';
import { tokens } from '@/theme/tokens';

interface MoodPickerProps {
  moods: readonly MoodOption[];
  selectedStatus: string | null;
  onSelect: (status: string) => void;
}

export function MoodPicker({
  moods,
  selectedStatus,
  onSelect,
}: MoodPickerProps) {
  return (
    <View accessibilityRole="radiogroup" style={styles.grid}>
      {moods.map((mood) => {
        const selected = mood.status === selectedStatus;
        return (
          <Pressable
            accessibilityLabel={mood.label}
            accessibilityRole="radio"
            accessibilityState={{ checked: selected }}
            key={mood.status}
            onPress={() => onSelect(mood.status)}
            style={({ pressed }) => [
              styles.option,
              selected && styles.optionSelected,
              pressed && styles.pressed,
            ]}
          >
            <Text style={[styles.symbol, selected && styles.selectedText]}>
              {mood.symbol}
            </Text>
            <Text style={[styles.label, selected && styles.selectedText]}>
              {mood.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  grid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: tokens.spacing.sm,
  },
  option: {
    width: '23%',
    minWidth: 112,
    minHeight: 92,
    alignItems: 'center',
    justifyContent: 'center',
    padding: tokens.spacing.sm,
    borderWidth: 1,
    borderColor: tokens.color.border,
    borderRadius: tokens.radius.md,
    backgroundColor: tokens.color.surfaceMuted,
  },
  optionSelected: {
    borderColor: tokens.color.highlight,
    borderWidth: 2,
    backgroundColor: tokens.color.accent,
  },
  pressed: { opacity: 0.8 },
  symbol: {
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingRegular,
    fontSize: 29,
  },
  label: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.text,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 14,
  },
  selectedText: { color: tokens.color.onAccent },
});
