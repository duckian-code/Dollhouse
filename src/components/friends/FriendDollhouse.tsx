import { Image, StyleSheet, Text, View } from 'react-native';

import { avatarImageSource, findAsset } from '@/services/avatar/catalog';
import { tokens } from '@/theme/tokens';
import type { AvatarCatalog, DollConfiguration } from '@/types/api';

interface FriendDollhouseProps {
  catalog: AvatarCatalog;
  doll: DollConfiguration;
  friendName: string;
}

export function FriendDollhouse({
  catalog,
  doll,
  friendName,
}: FriendDollhouseProps) {
  const layerIds = [
    doll.bodyAssetId,
    ...doll.clothingAssetIds,
    doll.eyesAssetId,
    doll.noseAssetId,
    doll.mouthAssetId,
    doll.hairAssetId,
  ];
  const layers = layerIds
    .map((assetId) => findAsset(catalog, assetId))
    .filter((asset) => asset !== undefined);

  return (
    <View
      accessible
      accessibilityLabel={`${friendName}'s four-room dollhouse and avatar`}
      style={styles.house}
    >
      <View style={[styles.room, styles.roomOne]}>
        <View style={styles.avatar}>
          {layers.map((asset, index) => (
            <Image
              accessibilityIgnoresInvertColors
              key={`${asset.assetId}-${index}`}
              resizeMode="contain"
              source={avatarImageSource(asset)}
              style={styles.layer}
            />
          ))}
          {!layers.length ? <Text style={styles.unavailable}>?</Text> : null}
        </View>
      </View>
      <View style={[styles.room, styles.roomTwo]}>
        <Text style={styles.furniture}>◇</Text>
      </View>
      <View style={[styles.room, styles.roomThree]}>
        <Text style={styles.furniture}>▱</Text>
      </View>
      <View style={[styles.room, styles.roomFour]}>
        <Text style={styles.furniture}>○</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  house: {
    width: '100%',
    maxWidth: 280,
    aspectRatio: 1,
    alignSelf: 'center',
    flexDirection: 'row',
    flexWrap: 'wrap',
    marginVertical: tokens.spacing.md,
    overflow: 'hidden',
    borderWidth: 4,
    borderColor: tokens.color.text,
    borderRadius: tokens.radius.sm,
    backgroundColor: tokens.color.surface,
  },
  room: {
    width: '50%',
    height: '50%',
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    borderColor: tokens.color.text,
  },
  roomOne: { backgroundColor: tokens.color.surfaceMuted },
  roomTwo: { backgroundColor: '#292938' },
  roomThree: { backgroundColor: '#352D3C' },
  roomFour: { backgroundColor: '#29343D' },
  avatar: { position: 'relative', width: '82%', aspectRatio: 1 },
  layer: { position: 'absolute', width: '100%', height: '100%' },
  unavailable: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.headingBold,
    fontSize: 32,
    textAlign: 'center',
  },
  furniture: {
    color: tokens.color.muted,
    fontFamily: tokens.typography.headingRegular,
    fontSize: 34,
  },
});
