export type ISODateTime = string;
export type UserId = string;
export type AssetId = string;

export type AssetCategory =
  'body' | 'hair' | 'eyes' | 'nose' | 'mouth' | 'clothing';

export interface AvatarAsset {
  assetId: AssetId;
  category: AssetCategory;
  url: string;
  contentType: 'image/png';
  width: number;
  height: number;
}

export interface AvatarCatalog {
  catalogVersion: string;
  expiresAt: ISODateTime;
  assets: AvatarAsset[];
}

export interface ApiResponse<T> {
  data: T;
}

export interface PaginatedResponse<T> {
  data: {
    items: T[];
    nextToken: string | null;
  };
}

export interface ErrorResponse {
  error: {
    code: string;
    message: string;
  };
}

export interface UserSummary {
  userId: UserId;
  username: string;
  displayName: string;
}

export interface Profile extends UserSummary {
  bio: string | null;
  onboardingComplete: boolean;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
}

export interface UpdateProfileRequest {
  username?: string;
  displayName?: string;
  bio?: string | null;
}

export interface DollConfiguration {
  bodyAssetId: AssetId;
  hairAssetId: AssetId;
  eyesAssetId: AssetId;
  noseAssetId: AssetId;
  mouthAssetId: AssetId;
  clothingAssetIds: AssetId[];
  updatedAt: ISODateTime;
}

export interface UpdateDollConfigurationRequest {
  bodyAssetId: AssetId;
  hairAssetId: AssetId;
  eyesAssetId: AssetId;
  noseAssetId: AssetId;
  mouthAssetId: AssetId;
  clothingAssetIds: AssetId[];
}

export type FriendshipStatus =
  'PENDING_INCOMING' | 'PENDING_OUTGOING' | 'ACCEPTED';

export interface FriendRequest {
  requestId: string;
  user: UserSummary;
  status: Extract<FriendshipStatus, 'PENDING_INCOMING' | 'PENDING_OUTGOING'>;
  requestedAt: ISODateTime;
}

export interface Friendship {
  friend: UserSummary;
  status: Extract<FriendshipStatus, 'ACCEPTED'>;
  acceptedAt: ISODateTime;
}

export interface SendFriendRequestRequest {
  userId: UserId;
}

export interface MoodState {
  status: string;
  stress: number | null;
  fatigue: number | null;
  discomfort: number | null;
  updatedAt: ISODateTime;
}

export interface PublishMoodRequest {
  status: string;
  stress?: number | null;
  fatigue?: number | null;
  discomfort?: number | null;
}

export interface FriendStatus {
  friend: UserSummary;
  doll: DollConfiguration;
  status: MoodState | null;
}

export type FriendStatusesResponse = PaginatedResponse<FriendStatus>;

export type ProfileResponse = ApiResponse<{ profile: Profile }>;
export type UsernameAvailabilityResponse = ApiResponse<{
  username: string;
  available: boolean;
}>;
export type AvatarCatalogResponse = ApiResponse<AvatarCatalog>;
export type DollConfigurationResponse = ApiResponse<{
  configuration: DollConfiguration;
}>;
export type FriendRequestResponse = ApiResponse<{
  friendRequest: FriendRequest;
}>;
export type FriendRequestsResponse = ApiResponse<{
  incoming: FriendRequest[];
  outgoing: FriendRequest[];
}>;
export type FriendshipResponse = ApiResponse<{ friendship: Friendship }>;
export type PublishMoodResponse = ApiResponse<{
  eventId: string;
  status: MoodState;
}>;
