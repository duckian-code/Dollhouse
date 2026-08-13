package profiledoll

import "context"

// Profile is the public representation of a Dollhouse user.
type Profile struct {
	UserID      string  `json:"userId" dynamodbav:"userId"`
	Username    string  `json:"username" dynamodbav:"username"`
	DisplayName string  `json:"displayName" dynamodbav:"displayName"`
	Bio         *string `json:"bio" dynamodbav:"bio"`
	CreatedAt   string  `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt   string  `json:"updatedAt" dynamodbav:"updatedAt"`
}

// DollConfiguration contains the approved asset IDs used to render a doll.
type DollConfiguration struct {
	BodyAssetID      string   `json:"bodyAssetId" dynamodbav:"bodyAssetId"`
	HairAssetID      string   `json:"hairAssetId" dynamodbav:"hairAssetId"`
	EyesAssetID      string   `json:"eyesAssetId" dynamodbav:"eyesAssetId"`
	NoseAssetID      string   `json:"noseAssetId" dynamodbav:"noseAssetId"`
	MouthAssetID     string   `json:"mouthAssetId" dynamodbav:"mouthAssetId"`
	ClothingAssetIDs []string `json:"clothingAssetIds" dynamodbav:"clothingAssetIds"`
	UpdatedAt        string   `json:"updatedAt" dynamodbav:"updatedAt"`
}

// Identity is derived exclusively from API Gateway's verified Cognito claims.
type Identity struct {
	UserID      string
	Username    string
	DisplayName string
}

// OptionalString distinguishes an omitted field from an explicit JSON null.
type OptionalString struct {
	Set   bool
	Value *string
}

// ProfileChanges contains only fields supplied by a profile update request.
type ProfileChanges struct {
	Username    OptionalString
	DisplayName OptionalString
	Bio         OptionalString
}

// Store is the persistence boundary used by the profile and doll handlers.
type Store interface {
	EnsureUser(context.Context, Identity, string) (Profile, error)
	UpdateProfile(context.Context, string, ProfileChanges, string) (Profile, error)
	GetDoll(context.Context, string) (*DollConfiguration, error)
	UpdateDoll(context.Context, string, DollConfiguration, string) (DollConfiguration, error)
}
