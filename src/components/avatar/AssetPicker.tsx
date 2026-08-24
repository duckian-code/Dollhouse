import { Image, Pressable, StyleSheet, Text, View } from 'react-native';

import { avatarImageSource } from '@/services/avatar/catalog';
import { tokens } from '@/theme/tokens';
import type { AvatarAsset } from '@/types/api';

type AssetPickerProps = {
  assets: AvatarAsset[];
  categoryLabel: string;
  onSelect: (assetId: string) => void;
  selectedIds: string[];
};

export function AssetPicker({
  assets,
  categoryLabel,
  onSelect,
  selectedIds,
}: AssetPickerProps) {
  if (!assets.length) {
    return (
      <Text style={styles.empty}>
        No {categoryLabel.toLowerCase()} available.
      </Text>
    );
  }

  return (
    <View accessibilityRole="radiogroup" style={styles.grid}>
      {assets.map((asset, index) => {
        const selected = selectedIds.includes(asset.assetId);
        return (
          <Pressable
            accessibilityLabel={`${categoryLabel} option ${index + 1}`}
            accessibilityRole="radio"
            accessibilityState={{ selected }}
            key={asset.assetId}
            onPress={() => onSelect(asset.assetId)}
            style={({ pressed }) => [
              styles.option,
              selected && styles.optionSelected,
              pressed && styles.pressed,
            ]}
          >
            <Image
              accessibilityIgnoresInvertColors
              resizeMode="contain"
              source={avatarImageSource(asset)}
              style={styles.image}
            />
            <Text numberOfLines={1} style={styles.optionLabel}>
              {asset.assetId}
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
    width: 82,
    minHeight: 98,
    alignItems: 'center',
    justifyContent: 'center',
    padding: tokens.spacing.sm,
    borderWidth: 1,
    borderColor: tokens.color.border,
    borderRadius: tokens.radius.md,
    backgroundColor: tokens.color.surface,
  },
  optionSelected: {
    borderWidth: 2,
    borderColor: tokens.color.accent,
    backgroundColor: tokens.color.surfaceMuted,
  },
  pressed: { opacity: 0.75 },
  image: { width: 58, height: 58 },
  optionLabel: {
    width: '100%',
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 12,
    textAlign: 'center',
  },
  empty: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 16,
  },
});
