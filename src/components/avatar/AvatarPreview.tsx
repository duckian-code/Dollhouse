import { useEffect, useState } from 'react';
import { Image, StyleSheet, Text, View } from 'react-native';

import { avatarImageSource, findAsset } from '@/services/avatar/catalog';
import { tokens } from '@/theme/tokens';
import type {
  AvatarCatalog,
  UpdateDollConfigurationRequest,
} from '@/types/api';

type AvatarPreviewProps = {
  catalog: AvatarCatalog;
  draft: UpdateDollConfigurationRequest;
  onUnavailableImagesChange?: (assetIds: string[]) => void;
  size?: number;
};

export function AvatarPreview({
  catalog,
  draft,
  onUnavailableImagesChange,
  size = 256,
}: AvatarPreviewProps) {
  const [failedIds, setFailedIds] = useState<string[]>([]);
  const layerIds = [
    draft.bodyAssetId,
    ...draft.clothingAssetIds,
    draft.eyesAssetId,
    draft.noseAssetId,
    draft.mouthAssetId,
    draft.hairAssetId,
  ];
  const activeFailedKey = failedIds
    .filter((assetId) => layerIds.includes(assetId))
    .join('\u0000');

  useEffect(() => {
    onUnavailableImagesChange?.(
      activeFailedKey ? activeFailedKey.split('\u0000') : [],
    );
  }, [activeFailedKey, onUnavailableImagesChange]);

  const visibleAssets = layerIds
    .map((assetId) => findAsset(catalog, assetId))
    .filter((asset) => asset !== undefined);

  return (
    <View
      accessibilityLabel="Doll preview"
      style={[styles.frame, { width: size, height: size }]}
    >
      <View style={styles.roomLine} />
      {visibleAssets.map((asset, index) =>
        failedIds.includes(asset.assetId) ? null : (
          <Image
            accessibilityIgnoresInvertColors
            key={`${asset.assetId}-${index}`}
            onError={() =>
              setFailedIds((current) =>
                current.includes(asset.assetId)
                  ? current
                  : [...current, asset.assetId],
              )
            }
            resizeMode="contain"
            source={avatarImageSource(asset)}
            style={styles.layer}
          />
        ),
      )}
      {!visibleAssets.length ? (
        <Text style={styles.empty}>Choose assets to preview your doll.</Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  frame: {
    position: 'relative',
    alignSelf: 'center',
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
    borderWidth: 1,
    borderColor: tokens.color.border,
    borderRadius: tokens.radius.lg,
    backgroundColor: tokens.color.surfaceMuted,
  },
  roomLine: {
    position: 'absolute',
    right: '10%',
    bottom: '18%',
    left: '10%',
    height: 1,
    backgroundColor: tokens.color.muted,
  },
  layer: {
    position: 'absolute',
    width: '100%',
    height: '100%',
  },
  empty: {
    padding: tokens.spacing.lg,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 17,
    textAlign: 'center',
  },
});
