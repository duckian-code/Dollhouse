import { router, type Href } from 'expo-router';
import { useEffect, useState } from 'react';
import {
  Modal,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';

import { AppCard } from '@/components/AppCard';
import { AppScreen } from '@/components/AppScreen';
import { LoadingState, RetryState } from '@/components/AsyncState';
import { FriendRow } from '@/components/friends/FriendRow';
import { SectionHeader } from '@/components/SectionHeader';
import { useFriendManagement } from '@/providers/FriendManagementProvider';
import { tokens } from '@/theme/tokens';
import type { UserSummary } from '@/types/api';

type FriendsView = 'friends' | 'requests';

export default function FriendsScreen() {
  const {
    friends,
    incoming,
    outgoing,
    isLoading,
    pendingActionId,
    error,
    isPreview,
    refresh,
    acceptRequest,
    declineRequest,
    removeFriend,
  } = useFriendManagement();
  const [view, setView] = useState<FriendsView>('friends');
  const [friendToRemove, setFriendToRemove] = useState<UserSummary | null>(
    null,
  );

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return (
    <AppScreen
      refreshControl={
        <RefreshControl
          refreshing={isLoading}
          onRefresh={() => void refresh()}
          tintColor={tokens.color.accent}
        />
      }
    >
      <SectionHeader
        title="Friends"
        subtitle="Find and manage the people in your Dollhouse."
      />
      {isPreview ? (
        <Text style={styles.previewNotice}>
          Local preview — friend changes update this session without contacting
          AWS.
        </Text>
      ) : null}
      {error ? (
        <RetryState
          message={error}
          onRetry={() => void refresh()}
          retrying={isLoading}
        />
      ) : null}
      <View accessibilityRole="tablist" style={styles.tabs}>
        <TabButton
          active={view === 'friends'}
          label="My friends"
          onPress={() => setView('friends')}
        />
        <TabButton
          active={view === 'requests'}
          label={`Requests (${incoming.length})`}
          onPress={() => setView('requests')}
        />
      </View>
      <Pressable
        accessibilityRole="button"
        onPress={() => router.push('/friend-search' as Href)}
        style={styles.findButton}
      >
        <Text style={styles.findButtonText}>＋ Find people</Text>
      </Pressable>

      {isLoading && !friends.length && !incoming.length ? (
        <LoadingState label="Opening your friends list…" />
      ) : view === 'friends' ? (
        <AppCard>
          <Text style={styles.sectionTitle}>Accepted friends</Text>
          {friends.length ? (
            friends.map((friend) => (
              <FriendRow
                actions={
                  <Pressable
                    accessibilityLabel={`Remove ${friend.displayName}`}
                    accessibilityRole="button"
                    disabled={pendingActionId === friend.userId}
                    onPress={() => setFriendToRemove(friend)}
                    style={styles.textButton}
                  >
                    <Text style={styles.removeText}>
                      {pendingActionId === friend.userId
                        ? 'Removing…'
                        : 'Remove'}
                    </Text>
                  </Pressable>
                }
                key={friend.userId}
                user={friend}
              />
            ))
          ) : (
            <EmptyState
              detail="Search for someone you trust and send the first request."
              title="No friends yet"
            />
          )}
        </AppCard>
      ) : (
        <View style={styles.requestSections}>
          <AppCard>
            <Text style={styles.sectionTitle}>Incoming requests</Text>
            {incoming.length ? (
              incoming.map((request) => (
                <FriendRow
                  actions={
                    <>
                      <Pressable
                        accessibilityLabel={`Decline ${request.user.displayName}`}
                        accessibilityRole="button"
                        disabled={pendingActionId === request.requestId}
                        onPress={() => void declineRequest(request)}
                        style={styles.smallButton}
                      >
                        <Text style={styles.secondaryAction}>Decline</Text>
                      </Pressable>
                      <Pressable
                        accessibilityLabel={`Accept ${request.user.displayName}`}
                        accessibilityRole="button"
                        disabled={pendingActionId === request.requestId}
                        onPress={() => void acceptRequest(request)}
                        style={[styles.smallButton, styles.acceptButton]}
                      >
                        <Text style={styles.acceptText}>
                          {pendingActionId === request.requestId
                            ? 'Working…'
                            : 'Accept'}
                        </Text>
                      </Pressable>
                    </>
                  }
                  detail={`Requested ${new Date(request.requestedAt).toLocaleDateString()}`}
                  key={request.requestId}
                  user={request.user}
                />
              ))
            ) : (
              <EmptyState
                detail="New requests will appear here."
                title="No incoming requests"
              />
            )}
          </AppCard>
          <AppCard>
            <Text style={styles.sectionTitle}>Sent requests</Text>
            {outgoing.length ? (
              outgoing.map((request) => (
                <FriendRow
                  detail="Waiting for a response"
                  key={request.requestId}
                  user={request.user}
                />
              ))
            ) : (
              <EmptyState
                detail="Requests you send will appear here."
                title="Nothing pending"
              />
            )}
          </AppCard>
        </View>
      )}

      <Modal
        animationType="fade"
        onRequestClose={() => setFriendToRemove(null)}
        transparent
        visible={friendToRemove !== null}
      >
        <View style={styles.modalBackdrop}>
          <View
            accessibilityViewIsModal
            accessibilityLabel="Remove friend confirmation"
            style={styles.modalCard}
          >
            <Text style={styles.modalTitle}>Remove this friend?</Text>
            <Text style={styles.modalBody}>
              {friendToRemove?.displayName} will no longer be able to see your
              shared status.
            </Text>
            <View style={styles.modalActions}>
              <Pressable
                accessibilityRole="button"
                onPress={() => setFriendToRemove(null)}
                style={styles.cancelButton}
              >
                <Text style={styles.cancelText}>Keep friend</Text>
              </Pressable>
              <Pressable
                accessibilityRole="button"
                onPress={() => {
                  if (!friendToRemove) return;
                  const target = friendToRemove;
                  setFriendToRemove(null);
                  void removeFriend(target);
                }}
                style={styles.confirmRemoveButton}
              >
                <Text style={styles.confirmRemoveText}>Remove</Text>
              </Pressable>
            </View>
          </View>
        </View>
      </Modal>
    </AppScreen>
  );
}

function TabButton({
  active,
  label,
  onPress,
}: {
  active: boolean;
  label: string;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="tab"
      accessibilityState={{ selected: active }}
      onPress={onPress}
      style={[styles.tab, active && styles.activeTab]}
    >
      <Text style={[styles.tabText, active && styles.activeTabText]}>
        {label}
      </Text>
    </Pressable>
  );
}

function EmptyState({ title, detail }: { title: string; detail: string }) {
  return (
    <View style={styles.emptyState}>
      <Text style={styles.emptyTitle}>{title}</Text>
      <Text style={styles.emptyDetail}>{detail}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  previewNotice: {
    marginBottom: tokens.spacing.md,
    padding: tokens.spacing.sm,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 15,
    textAlign: 'center',
    borderRadius: tokens.radius.sm,
    backgroundColor: tokens.color.surfaceMuted,
  },
  errorState: {
    marginBottom: tokens.spacing.md,
    padding: tokens.spacing.md,
    borderWidth: 1,
    borderColor: tokens.color.secondaryAccent,
    borderRadius: tokens.radius.sm,
  },
  errorText: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.bodySemibold,
    fontSize: 16,
  },
  retryText: {
    marginTop: tokens.spacing.xs,
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingBold,
  },
  tabs: { flexDirection: 'row', gap: tokens.spacing.sm },
  tab: {
    flex: 1,
    minHeight: 44,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.surface,
  },
  activeTab: { backgroundColor: tokens.color.accent },
  tabText: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 14,
  },
  activeTabText: { color: tokens.color.onAccent },
  findButton: {
    minHeight: 50,
    marginVertical: tokens.spacing.md,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.surfaceMuted,
  },
  findButtonText: {
    color: tokens.color.highlight,
    fontFamily: tokens.typography.headingBold,
    fontSize: 16,
  },
  loader: { marginTop: tokens.spacing.xl },
  sectionTitle: {
    marginBottom: tokens.spacing.sm,
    color: tokens.color.text,
    fontFamily: tokens.typography.headingBold,
    fontSize: 19,
  },
  requestSections: { gap: tokens.spacing.md },
  textButton: {
    minHeight: 40,
    justifyContent: 'center',
    paddingHorizontal: tokens.spacing.sm,
  },
  removeText: {
    color: tokens.color.secondaryAccent,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 14,
  },
  smallButton: {
    minHeight: 38,
    justifyContent: 'center',
    paddingHorizontal: tokens.spacing.sm,
    borderWidth: 1,
    borderColor: tokens.color.border,
    borderRadius: tokens.radius.round,
  },
  acceptButton: {
    borderColor: tokens.color.accent,
    backgroundColor: tokens.color.accent,
  },
  secondaryAction: {
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.headingSemibold,
    fontSize: 13,
  },
  acceptText: {
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
  modalBackdrop: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: tokens.spacing.lg,
    backgroundColor: 'rgba(0,0,0,0.72)',
  },
  modalCard: {
    width: '100%',
    maxWidth: 460,
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
  modalBody: {
    marginTop: tokens.spacing.sm,
    color: tokens.color.textMuted,
    fontFamily: tokens.typography.bodyRegular,
    fontSize: 17,
  },
  modalActions: {
    flexDirection: 'row',
    gap: tokens.spacing.sm,
    marginTop: tokens.spacing.lg,
  },
  cancelButton: {
    flex: 1,
    minHeight: 48,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    borderColor: tokens.color.border,
    borderRadius: tokens.radius.round,
  },
  cancelText: {
    color: tokens.color.text,
    fontFamily: tokens.typography.headingSemibold,
  },
  confirmRemoveButton: {
    flex: 1,
    minHeight: 48,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: tokens.radius.round,
    backgroundColor: tokens.color.secondaryAccent,
  },
  confirmRemoveText: {
    color: tokens.color.background,
    fontFamily: tokens.typography.headingBold,
  },
});
