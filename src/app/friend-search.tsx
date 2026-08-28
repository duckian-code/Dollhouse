import { router } from 'expo-router';
import { useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { FriendRow } from '@/components/friends/FriendRow';
import { SectionHeader } from '@/components/SectionHeader';
import { useFriendManagement } from '@/providers/FriendManagementProvider';
import { tokens } from '@/theme/tokens';

export default function FriendSearchScreen() {
  const {
    searchResults,
    outgoing,
    isSearching,
    pendingActionId,
    error,
    isPreview,
    search,
    sendRequest,
    clearError,
  } = useFriendManagement();
  const [query, setQuery] = useState('');
  const [hasSearched, setHasSearched] = useState(false);

  function submitSearch() {
    setHasSearched(true);
    void search(query);
  }

  return (
    <AppScreen>
      <View style={styles.toolbar}>
        <Pressable
          accessibilityRole="button"
          onPress={() => router.back()}
          style={styles.backButton}
        >
          <Text style={styles.backText}>‹ Friends</Text>
        </Pressable>
      </View>
      <SectionHeader
        title="Find people"
        subtitle="Search by the beginning of someone’s username."
      />
      {isPreview ? (
        <Text style={styles.previewNotice}>Local preview search</Text>
      ) : null}
      <View style={styles.searchRow}>
        <TextInput
          accessibilityLabel="Username search"
          autoCapitalize="none"
          autoCorrect={false}
          maxLength={50}
          onChangeText={(value) => {
            setQuery(value);
            clearError();
          }}
          onSubmitEditing={submitSearch}
          placeholder="Username"
          placeholderTextColor={tokens.color.muted}
          returnKeyType="search"
          style={styles.input}
          value={query}
        />
        <Pressable
          accessibilityRole="button"
          disabled={isSearching}
          onPress={submitSearch}
          style={styles.searchButton}
        >
          <Text style={styles.searchButtonText}>Search</Text>
        </Pressable>
      </View>
      {error ? (
        <Text accessibilityLiveRegion="assertive" style={styles.error}>
          {error}
        </Text>
      ) : null}
      {isSearching ? (
        <ActivityIndicator color={tokens.color.accent} style={styles.loader} />
      ) : hasSearched ? (
        <AppCard>
          <Text style={styles.sectionTitle}>Search results</Text>
          {searchResults.length ? (
            searchResults.map((user) => {
              const pending = outgoing.some(
                (request) => request.user.userId === user.userId,
              );
              return (
                <FriendRow
                  actions={
                    <Pressable
                      accessibilityLabel={`Send friend request to ${user.displayName}`}
                      accessibilityRole="button"
                      disabled={pending || pendingActionId === user.userId}
                      onPress={() => void sendRequest(user)}
                      style={[
                        styles.addButton,
                        pending && styles.pendingButton,
                      ]}
                    >
                      <Text style={styles.addButtonText}>
                        {pending
                          ? 'Pending'
                          : pendingActionId === user.userId
                            ? 'Sending…'
                            : 'Add'}
                      </Text>
                    </Pressable>
                  }
                  key={user.userId}
                  user={user}
                />
              );
            })
          ) : (
            <View style={styles.emptyState}>
              <Text style={styles.emptyTitle}>No users found</Text>
              <Text style={styles.emptyDetail}>
                Check the username and try another search.
              </Text>
            </View>
          )}
        </AppCard>
      ) : (
        <Text style={styles.prompt}>
          Enter at least one character to begin.
        </Text>
      )}
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  toolbar: { minHeight: 44, justifyContent: 'center' },
  backButton: {
    minHeight: 44,
    alignSelf: 'flex-start',
    justifyContent: 'center',
  },
  backText: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 16,
  },
  previewNotice: {
    marginBottom: tokens.spacing.md,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 15,
  },
  searchRow: { flexDirection: 'row', gap: tokens.spacing.sm },
  input: {
    flex: 1,
    minHeight: 50,
    paddingHorizontal: tokens.spacing.md,
    color: tokens.color.text,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 17,
    borderWidth: 1,
    borderColor: tokens.color.border,
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.surface,
  },
  searchButton: {
    minWidth: 96,
    minHeight: 50,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  searchButtonText: {
    color: tokens.color.onAccent,
    fontFamily: tokens.typography.headingBold,
  },
  error: {
    marginTop: tokens.spacing.md,
    padding: tokens.spacing.md,
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.bodySemibold,
    borderWidth: 1,
    borderColor: tokens.color.secondaryAccent,
    borderRadius: tokens.radius.sm,
  },
  loader: { marginTop: tokens.spacing.xl },
  sectionTitle: {
    marginBottom: tokens.spacing.sm,
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 19,
  },
  addButton: {
    minWidth: 72,
    minHeight: 40,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: tokens.spacing.sm,
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.accent,
  },
  pendingButton: { opacity: 0.55 },
  addButtonText: {
    color: tokens.color.onAccent,
    fontFamily: tokens.typography.headingBold,
    fontSize: 13,
  },
  emptyState: { alignItems: 'center', paddingVertical: tokens.spacing.xl },
  emptyTitle: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 17,
  },
  emptyDetail: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 15,
    textAlign: 'center',
  },
  prompt: {
    marginTop: tokens.spacing.lg,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 16,
    textAlign: 'center',
  },
});
