import type { ImageSourcePropType } from 'react-native';

import body00 from '@/assets/avatar/body-00.png';
import eyes00 from '@/assets/avatar/eyes-00.png';
import hair00 from '@/assets/avatar/hair-00.png';
import hair02 from '@/assets/avatar/hair-02.png';
import mouth00 from '@/assets/avatar/mouth-00.png';
import nose00 from '@/assets/avatar/nose-00.png';
import type {
  AssetCategory,
  AvatarAsset,
  AvatarCatalog,
  DollConfiguration,
  UpdateDollConfigurationRequest,
} from '@/types/api';

const localSources: Record<string, ImageSourcePropType> = {
  'body-00': body00,
  'eyes-00': eyes00,
  'hair-00': hair00,
  'hair-02': hair02,
  'mouth-00': mouth00,
  'nose-00': nose00,
};

const localAssets: AvatarAsset[] = [
  ['body-00', 'body'],
  ['eyes-00', 'eyes'],
  ['hair-00', 'hair'],
  ['hair-02', 'hair'],
  ['mouth-00', 'mouth'],
  ['nose-00', 'nose'],
].map(([assetId, category]) => ({
  assetId,
  category: category as AssetCategory,
  url: '',
  contentType: 'image/png',
  width: 64,
  height: 64,
}));

export const developmentAvatarCatalog: AvatarCatalog = {
  catalogVersion: 'local-development-v1',
  expiresAt: '9999-12-31T23:59:59Z',
  assets: localAssets,
};

export function avatarImageSource(asset: AvatarAsset): ImageSourcePropType {
  return asset.url
    ? { uri: asset.url }
    : (localSources[asset.assetId] ?? { uri: '' });
}

export function assetsByCategory(
  catalog: AvatarCatalog,
  category: AssetCategory,
) {
  return catalog.assets.filter((asset) => asset.category === category);
}

export function findAsset(catalog: AvatarCatalog, assetId: string) {
  return catalog.assets.find((asset) => asset.assetId === assetId);
}

export function createDefaultDraft(
  catalog: AvatarCatalog,
): UpdateDollConfigurationRequest {
  const firstId = (category: AssetCategory) =>
    assetsByCategory(catalog, category)[0]?.assetId ?? '';
  return {
    bodyAssetId: firstId('body'),
    hairAssetId: firstId('hair'),
    eyesAssetId: firstId('eyes'),
    noseAssetId: firstId('nose'),
    mouthAssetId: firstId('mouth'),
    clothingAssetIds: [],
  };
}

export function configurationDraft(
  configuration: DollConfiguration,
): UpdateDollConfigurationRequest {
  const { updatedAt: _updatedAt, ...draft } = configuration;
  return draft;
}

export function unavailableDraftIds(
  catalog: AvatarCatalog,
  draft: UpdateDollConfigurationRequest,
) {
  const catalogIds = new Set(catalog.assets.map(({ assetId }) => assetId));
  return [
    draft.bodyAssetId,
    draft.hairAssetId,
    draft.eyesAssetId,
    draft.noseAssetId,
    draft.mouthAssetId,
    ...draft.clothingAssetIds,
  ].filter((assetId) => assetId && !catalogIds.has(assetId));
}

export function draftIsComplete(draft: UpdateDollConfigurationRequest) {
  return Boolean(
    draft.bodyAssetId &&
    draft.hairAssetId &&
    draft.eyesAssetId &&
    draft.noseAssetId &&
    draft.mouthAssetId,
  );
}
