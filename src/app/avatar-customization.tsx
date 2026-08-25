import { router } from 'expo-router';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';

import { AppScreen } from '@/components/AppScreen';
import { AssetPicker } from '@/components/avatar/AssetPicker';
import { AvatarPreview } from '@/components/avatar/AvatarPreview';
import { SectionHeader } from '@/components/SectionHeader';
import { ApiError } from '@/services/api/client';
import {
  getDollConfiguration,
  updateDollConfiguration,
} from '@/services/api/dollhouse';
import {
  assetsByCategory,
  configurationDraft,
  createDefaultDraft,
  developmentAvatarCatalog,
  draftIsComplete,
  unavailableDraftIds,
} from '@/services/avatar/catalog';
import { loadAvatarCatalog } from '@/services/cache/dollhouse';
import { tokens } from '@/theme/tokens';
import type {
  AssetCategory,
  AvatarCatalog,
  UpdateDollConfigurationRequest,
} from '@/types/api';

type EditorTab = 'body' | 'hair' | 'face' | 'clothing';

const tabs: { label: string; value: EditorTab }[] = [
  { label: 'Body', value: 'body' },
  { label: 'Hair', value: 'hair' },
  { label: 'Face & eyes', value: 'face' },
  { label: 'Clothing', value: 'clothing' },
];

export default function AvatarCustomizationScreen() {
  const [catalog, setCatalog] = useState<AvatarCatalog>();
  const [draft, setDraft] = useState<UpdateDollConfigurationRequest>();
  const [initialDraft, setInitialDraft] =
    useState<UpdateDollConfigurationRequest>();
  const [activeTab, setActiveTab] = useState<EditorTab>('hair');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [fixtureMode, setFixtureMode] = useState(false);
  const [failedImageIds, setFailedImageIds] = useState<string[]>([]);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const loadEditor = useCallback(async () => {
    setLoading(true);
    setError('');
    setNotice('');
    setFailedImageIds([]);

    try {
      const catalogRequest = await loadAvatarCatalog();
      if (catalogRequest.cached && !catalogRequest.cached.isStale) {
        setCatalog(catalogRequest.cached.data);
      }

      let resolvedCatalog: AvatarCatalog;
      let usingFixture = false;
      try {
        resolvedCatalog = (await catalogRequest.refresh).data;
        setFixtureMode(false);
      } catch (caught) {
        if (catalogRequest.cached && !catalogRequest.cached.isStale) {
          resolvedCatalog = catalogRequest.cached.data;
          setNotice('Showing the recently cached avatar catalog.');
        } else if (__DEV__) {
          resolvedCatalog = developmentAvatarCatalog;
          usingFixture = true;
          setFixtureMode(true);
          setNotice(
            'Using the local starter assets. Saving requires the live avatar catalog.',
          );
        } else {
          throw caught;
        }
      }
      setCatalog(resolvedCatalog);

      let restoredDraft: UpdateDollConfigurationRequest;
      try {
        const response = await getDollConfiguration();
        restoredDraft = configurationDraft(response.data.configuration);
      } catch (caught) {
        if (caught instanceof ApiError && caught.status === 404) {
          restoredDraft = createDefaultDraft(resolvedCatalog);
        } else if (__DEV__ && usingFixture) {
          restoredDraft = createDefaultDraft(resolvedCatalog);
        } else {
          throw caught;
        }
      }
      setDraft(restoredDraft);
      setInitialDraft(restoredDraft);
    } catch (caught) {
      setError(
        caught instanceof Error
          ? caught.message
          : 'The avatar editor could not be loaded.',
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeout = setTimeout(() => void loadEditor(), 0);
    return () => clearTimeout(timeout);
  }, [loadEditor]);

  const unavailableIds = useMemo(
    () => (catalog && draft ? unavailableDraftIds(catalog, draft) : []),
    [catalog, draft],
  );
  const dirty = Boolean(
    draft &&
    initialDraft &&
    JSON.stringify(draft) !== JSON.stringify(initialDraft),
  );

  function selectSingle(
    field: Exclude<keyof UpdateDollConfigurationRequest, 'clothingAssetIds'>,
    assetId: string,
  ) {
    setDraft((current) =>
      current ? { ...current, [field]: assetId } : current,
    );
    setNotice('');
  }

  function toggleClothing(assetId: string) {
    setDraft((current) => {
      if (!current) return current;
      const selected = current.clothingAssetIds.includes(assetId);
      return {
        ...current,
        clothingAssetIds: selected
          ? current.clothingAssetIds.filter((id) => id !== assetId)
          : [...current.clothingAssetIds, assetId],
      };
    });
    setNotice('');
  }

  async function save() {
    if (!draft || fixtureMode) return;
    setSaving(true);
    setError('');
    setNotice('');
    try {
      const response = await updateDollConfiguration(draft);
      const savedDraft = configurationDraft(response.data.configuration);
      setDraft(savedDraft);
      setInitialDraft(savedDraft);
      setNotice('Your doll customization was saved.');
    } catch (caught) {
      setError(
        caught instanceof Error
          ? caught.message
          : 'Your customization could not be saved.',
      );
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <AppScreen scroll={false} style={styles.centered}>
        <ActivityIndicator color={tokens.color.accent} size="large" />
        <Text style={styles.loadingText}>Preparing your avatar…</Text>
      </AppScreen>
    );
  }

  if (!catalog || !draft) {
    return (
      <AppScreen contentContainerStyle={styles.content}>
        <SectionHeader
          title="Avatar unavailable"
          subtitle={error || 'The avatar editor could not be loaded.'}
        />
        <ActionButton label="Try again" onPress={loadEditor} />
      </AppScreen>
    );
  }

  const canSave =
    !fixtureMode &&
    dirty &&
    draftIsComplete(draft) &&
    unavailableIds.length === 0 &&
    failedImageIds.length === 0 &&
    !saving;

  return (
    <AppScreen contentContainerStyle={styles.content}>
      <View style={styles.toolbar}>
        <Pressable accessibilityRole="button" onPress={() => router.back()}>
          <Text style={styles.toolbarLink}>Cancel</Text>
        </Pressable>
        <Pressable
          accessibilityRole="button"
          disabled={!dirty}
          onPress={() => initialDraft && setDraft(initialDraft)}
        >
          <Text style={[styles.toolbarLink, !dirty && styles.disabledText]}>
            Reset
          </Text>
        </Pressable>
      </View>
      <SectionHeader
        title="Customize your doll"
        subtitle="Choose each layer, preview the result, and save it to your Dollhouse."
      />

      <AvatarPreview
        catalog={catalog}
        draft={draft}
        key={catalog.catalogVersion}
        onUnavailableImagesChange={setFailedImageIds}
      />

      {notice ? <Text style={styles.notice}>{notice}</Text> : null}
      {error ? <Text style={styles.error}>{error}</Text> : null}
      {unavailableIds.length ? (
        <Text style={styles.error}>
          Replace unavailable saved assets: {unavailableIds.join(', ')}
        </Text>
      ) : null}
      {failedImageIds.length ? (
        <View style={styles.inlineMessage}>
          <Text style={styles.errorText}>
            Some images could not be displayed: {failedImageIds.join(', ')}
          </Text>
          <Pressable accessibilityRole="button" onPress={loadEditor}>
            <Text style={styles.inlineLink}>Refresh catalog</Text>
          </Pressable>
        </View>
      ) : null}

      <View accessibilityRole="tablist" style={styles.tabs}>
        {tabs.map((tab) => (
          <Pressable
            accessibilityRole="tab"
            accessibilityState={{ selected: activeTab === tab.value }}
            key={tab.value}
            onPress={() => setActiveTab(tab.value)}
            style={[styles.tab, activeTab === tab.value && styles.tabActive]}
          >
            <Text
              style={[
                styles.tabLabel,
                activeTab === tab.value && styles.tabLabelActive,
              ]}
            >
              {tab.label}
            </Text>
          </Pressable>
        ))}
      </View>

      <View style={styles.pickerPanel}>
        {activeTab === 'body' ? (
          <PickerSection
            catalog={catalog}
            category="body"
            label="Body"
            onSelect={(id) => selectSingle('bodyAssetId', id)}
            selectedIds={[draft.bodyAssetId]}
          />
        ) : null}
        {activeTab === 'hair' ? (
          <PickerSection
            catalog={catalog}
            category="hair"
            label="Hair"
            onSelect={(id) => selectSingle('hairAssetId', id)}
            selectedIds={[draft.hairAssetId]}
          />
        ) : null}
        {activeTab === 'face' ? (
          <>
            <PickerSection
              catalog={catalog}
              category="eyes"
              label="Eyes"
              onSelect={(id) => selectSingle('eyesAssetId', id)}
              selectedIds={[draft.eyesAssetId]}
            />
            <PickerSection
              catalog={catalog}
              category="nose"
              label="Nose"
              onSelect={(id) => selectSingle('noseAssetId', id)}
              selectedIds={[draft.noseAssetId]}
            />
            <PickerSection
              catalog={catalog}
              category="mouth"
              label="Mouth"
              onSelect={(id) => selectSingle('mouthAssetId', id)}
              selectedIds={[draft.mouthAssetId]}
            />
          </>
        ) : null}
        {activeTab === 'clothing' ? (
          <PickerSection
            catalog={catalog}
            category="clothing"
            label="Clothing"
            onSelect={toggleClothing}
            selectedIds={draft.clothingAssetIds}
          />
        ) : null}
      </View>

      <ActionButton
        disabled={!canSave}
        label={saving ? 'Saving…' : 'Save customization'}
        onPress={save}
      />
    </AppScreen>
  );
}

function PickerSection({
  catalog,
  category,
  label,
  onSelect,
  selectedIds,
}: {
  catalog: AvatarCatalog;
  category: AssetCategory;
  label: string;
  onSelect: (assetId: string) => void;
  selectedIds: string[];
}) {
  return (
    <View style={styles.pickerSection}>
      <Text style={styles.pickerTitle}>{label}</Text>
      <AssetPicker
        assets={assetsByCategory(catalog, category)}
        categoryLabel={label}
        onSelect={onSelect}
        selectedIds={selectedIds}
      />
    </View>
  );
}

function ActionButton({
  disabled = false,
  label,
  onPress,
}: {
  disabled?: boolean;
  label: string;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.button,
        disabled && styles.buttonDisabled,
        pressed && styles.pressed,
      ]}
    >
      <Text style={styles.buttonText}>{label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  content: { maxWidth: 760 },
  centered: { alignItems: 'center', justifyContent: 'center' },
  loadingText: {
    marginTop: tokens.spacing.md,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 18,
  },
  toolbar: {
    minHeight: 44,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  toolbarLink: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 16,
  },
  disabledText: { color: tokens.color.muted },
  notice: {
    marginTop: tokens.spacing.md,
    padding: tokens.spacing.md,
    borderRadius: tokens.radius.sm,
    backgroundColor: tokens.color.surface,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 16,
  },
  error: {
    marginTop: tokens.spacing.md,
    padding: tokens.spacing.md,
    borderWidth: 1,
    borderColor: tokens.color.secondaryAccent,
    borderRadius: tokens.radius.sm,
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 16,
  },
  inlineMessage: { gap: tokens.spacing.sm },
  errorText: {
    marginTop: tokens.spacing.md,
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 16,
  },
  inlineLink: {
    color: tokens.color.accent,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 15,
  },
  tabs: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: tokens.spacing.sm,
    marginTop: tokens.spacing.lg,
  },
  tab: {
    minHeight: 42,
    justifyContent: 'center',
    paddingHorizontal: tokens.spacing.md,
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.surface,
  },
  tabActive: { backgroundColor: tokens.color.accent },
  tabLabel: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 14,
  },
  tabLabelActive: { color: tokens.color.onAccent },
  pickerPanel: {
    gap: tokens.spacing.lg,
    marginTop: tokens.spacing.md,
    padding: tokens.spacing.md,
    borderRadius: tokens.radius.lg,
    backgroundColor: tokens.color.surfaceMuted,
  },
  pickerSection: { gap: tokens.spacing.sm },
  pickerTitle: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 18,
  },
  button: {
    minHeight: 54,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: tokens.spacing.lg,
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  buttonDisabled: { opacity: 0.4 },
  pressed: { opacity: 0.75 },
  buttonText: {
    color: tokens.color.onAccent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 16,
  },
});
