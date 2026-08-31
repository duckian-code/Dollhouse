package profiledoll

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/dollhouse-app/dollhouse/backend/internal/assetcatalog"
)

type fakeStore struct {
	profile       Profile
	doll          *DollConfiguration
	err           error
	ensured       []Identity
	profileUserID string
	changes       ProfileChanges
	getDollUserID string
	dollUserID    string
}

type fakeAssetValidator struct {
	references []assetcatalog.Reference
	err        error
}

func (v *fakeAssetValidator) Validate(_ context.Context, references []assetcatalog.Reference) error {
	v.references = references
	return v.err
}

func (s *fakeStore) EnsureUser(_ context.Context, identity Identity, _ string) (Profile, error) {
	s.ensured = append(s.ensured, identity)
	return s.profile, s.err
}

func (s *fakeStore) UpdateProfile(_ context.Context, userID string, changes ProfileChanges, _ string) (Profile, error) {
	s.profileUserID, s.changes = userID, changes
	return s.profile, s.err
}

func (s *fakeStore) GetDoll(_ context.Context, userID string) (*DollConfiguration, error) {
	s.getDollUserID = userID
	return s.doll, s.err
}

func (s *fakeStore) UpdateDoll(_ context.Context, userID string, configuration DollConfiguration, now string) (DollConfiguration, error) {
	s.dollUserID = userID
	configuration.UpdatedAt = now
	return configuration, s.err
}

func authenticatedRequest(body string) events.APIGatewayV2HTTPRequest {
	request := events.APIGatewayV2HTTPRequest{Body: body}
	request.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
		JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: map[string]string{
			"sub": "user-123", "username": "alice", "name": "Alice",
		}},
	}
	return request
}

func fixedHandlers(store Store) *Handlers {
	h := NewHandlers(store, &fakeAssetValidator{})
	h.now = func() time.Time { return time.Date(2026, 8, 13, 12, 30, 45, 0, time.FixedZone("test", -6*60*60)) }
	return h
}

func TestGetProfileUsesAuthenticatedIdentity(t *testing.T) {
	store := &fakeStore{profile: Profile{UserID: "user-123", Username: "alice", DisplayName: "Alice", CreatedAt: "2026-08-13T18:30:45Z", UpdatedAt: "2026-08-13T18:30:45Z"}}
	got, err := fixedHandlers(store).GetProfile(context.Background(), authenticatedRequest(""))
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 200 {
		t.Fatalf("status = %d, body = %s", got.StatusCode, got.Body)
	}
	if len(store.ensured) != 1 || store.ensured[0].UserID != "user-123" {
		t.Fatalf("ensured = %#v", store.ensured)
	}
	var body struct {
		Data struct {
			Profile Profile `json:"profile"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Profile.DisplayName != "Alice" {
		t.Fatalf("profile = %#v", body.Data.Profile)
	}
}

func TestGetProfileRequiresVerifiedClaims(t *testing.T) {
	got, err := fixedHandlers(&fakeStore{}).GetProfile(context.Background(), events.APIGatewayV2HTTPRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 401 || !jsonErrorCode(got.Body, "unauthenticated") {
		t.Fatalf("response = %#v", got)
	}
}

func TestEveryProfileAndDollRouteRequiresAuthentication(t *testing.T) {
	handlers := fixedHandlers(&fakeStore{})
	tests := map[string]func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error){
		"GET /profile": handlers.GetProfile,
		"PUT /profile": handlers.UpdateProfile,
		"GET /doll":    handlers.GetDoll,
		"PUT /doll":    handlers.UpdateDoll,
	}
	for name, invoke := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := invoke(context.Background(), events.APIGatewayV2HTTPRequest{})
			if err != nil || got.StatusCode != http.StatusUnauthorized || !jsonErrorCode(got.Body, "unauthenticated") {
				t.Fatalf("response = %#v, err = %v", got, err)
			}
		})
	}
}

func TestUpdateProfileAppliesOnlySuppliedFields(t *testing.T) {
	bio := "old bio"
	store := &fakeStore{profile: Profile{UserID: "user-123", Username: "alice", DisplayName: "Alice Updated", Bio: &bio}}
	got, err := fixedHandlers(store).UpdateProfile(context.Background(), authenticatedRequest(`{"displayName":" Alice Updated ","bio":null}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 200 {
		t.Fatalf("status = %d, body = %s", got.StatusCode, got.Body)
	}
	if store.profileUserID != "user-123" {
		t.Fatalf("updated user = %q", store.profileUserID)
	}
	if store.changes.Username.Set {
		t.Fatal("omitted username was marked for update")
	}
	if !store.changes.DisplayName.Set || *store.changes.DisplayName.Value != "Alice Updated" {
		t.Fatalf("display change = %#v", store.changes.DisplayName)
	}
	if !store.changes.Bio.Set || store.changes.Bio.Value != nil {
		t.Fatalf("bio change = %#v", store.changes.Bio)
	}
}

func TestUpdateProfileRejectsUnknownAndEmptyUpdates(t *testing.T) {
	tests := []string{`{}`, `{"email":"alice@example.test"}`, `{"displayName":null}`}
	for _, body := range tests {
		got, err := fixedHandlers(&fakeStore{}).UpdateProfile(context.Background(), authenticatedRequest(body))
		if err != nil {
			t.Fatal(err)
		}
		if got.StatusCode != 400 {
			t.Errorf("body %s: status = %d, response = %s", body, got.StatusCode, got.Body)
		}
	}
}

func TestGetDollReturnsNotFoundWhenNotConfigured(t *testing.T) {
	store := &fakeStore{}
	got, err := fixedHandlers(store).GetDoll(context.Background(), authenticatedRequest(""))
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 404 || !jsonErrorCode(got.Body, "not_found") {
		t.Fatalf("response = %#v", got)
	}
	if store.getDollUserID != "user-123" {
		t.Fatalf("read user = %q", store.getDollUserID)
	}
}

func TestUpdateDollReplacesConfigurationForAuthenticatedUser(t *testing.T) {
	store := &fakeStore{}
	assets := &fakeAssetValidator{}
	body := `{"bodyAssetId":"body-1","hairAssetId":"hair-1","eyesAssetId":"eyes-1","noseAssetId":"nose-1","mouthAssetId":"mouth-1","clothingAssetIds":["shirt-1"]}`
	handlers := NewHandlers(store, assets)
	handlers.now = func() time.Time { return time.Date(2026, 8, 13, 12, 30, 45, 0, time.FixedZone("test", -6*60*60)) }
	got, err := handlers.UpdateDoll(context.Background(), authenticatedRequest(body))
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 200 {
		t.Fatalf("status = %d, body = %s", got.StatusCode, got.Body)
	}
	if store.dollUserID != "user-123" {
		t.Fatalf("updated user = %q", store.dollUserID)
	}
	if len(assets.references) != 6 || assets.references[0] != (assetcatalog.Reference{Field: "bodyAssetId", AssetID: "body-1", Category: "body"}) || assets.references[5] != (assetcatalog.Reference{Field: "clothingAssetIds[0]", AssetID: "shirt-1", Category: "clothing"}) {
		t.Fatalf("asset references = %#v", assets.references)
	}
	var responseBody struct {
		Data struct {
			Configuration DollConfiguration `json:"configuration"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(got.Body), &responseBody); err != nil {
		t.Fatal(err)
	}
	if responseBody.Data.Configuration.UpdatedAt != "2026-08-13T18:30:45Z" {
		t.Fatalf("configuration = %#v", responseBody.Data.Configuration)
	}
}

func TestUpdateDollRejectsUnapprovedAssetBeforeWriting(t *testing.T) {
	store := &fakeStore{}
	assets := &fakeAssetValidator{err: &assetcatalog.SelectionError{Message: `bodyAssetId references unapproved asset ID "body-unknown"`}}
	body := `{"bodyAssetId":"body-unknown","hairAssetId":"hair-1","eyesAssetId":"eyes-1","noseAssetId":"nose-1","mouthAssetId":"mouth-1","clothingAssetIds":[]}`
	got, err := NewHandlers(store, assets).UpdateDoll(context.Background(), authenticatedRequest(body))
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 400 || !jsonErrorCode(got.Body, "validation_failed") {
		t.Fatalf("response = %#v", got)
	}
	if store.dollUserID != "" || len(store.ensured) != 0 {
		t.Fatalf("invalid configuration reached storage: store = %#v", store)
	}
}

func TestUpdateDollDoesNotExposeCatalogFailures(t *testing.T) {
	assets := &fakeAssetValidator{err: errors.New("secret S3 detail")}
	body := `{"bodyAssetId":"body-1","hairAssetId":"hair-1","eyesAssetId":"eyes-1","noseAssetId":"nose-1","mouthAssetId":"mouth-1","clothingAssetIds":[]}`
	got, err := NewHandlers(&fakeStore{}, assets).UpdateDoll(context.Background(), authenticatedRequest(body))
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 500 || !jsonErrorCode(got.Body, "internal_error") || contains(got.Body, "secret S3 detail") {
		t.Fatalf("response = %#v", got)
	}
}

func TestUpdateDollValidatesFullConfiguration(t *testing.T) {
	tests := []string{
		`{"hairAssetId":"hair-1","eyesAssetId":"eyes-1","noseAssetId":"nose-1","mouthAssetId":"mouth-1","clothingAssetIds":[]}`,
		`{"bodyAssetId":"body-1","hairAssetId":"hair-1","eyesAssetId":"eyes-1","noseAssetId":"nose-1","mouthAssetId":"mouth-1"}`,
		`{"bodyAssetId":"body-1","hairAssetId":"hair-1","eyesAssetId":"eyes-1","noseAssetId":"nose-1","mouthAssetId":"mouth-1","clothingAssetIds":["shirt-1","shirt-1"]}`,
	}
	for _, body := range tests {
		got, err := fixedHandlers(&fakeStore{}).UpdateDoll(context.Background(), authenticatedRequest(body))
		if err != nil {
			t.Fatal(err)
		}
		if got.StatusCode != 400 || !jsonErrorCode(got.Body, "validation_failed") {
			t.Errorf("body %s: response = %#v", body, got)
		}
	}
}

func TestStorageErrorsAreNotExposed(t *testing.T) {
	store := &fakeStore{err: errors.New("secret database detail")}
	got, err := fixedHandlers(store).GetProfile(context.Background(), authenticatedRequest(""))
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 500 || !jsonErrorCode(got.Body, "internal_error") {
		t.Fatalf("response = %#v", got)
	}
	if got.Body == "" || contains(got.Body, "secret database detail") {
		t.Fatalf("internal detail leaked: %s", got.Body)
	}
}

func jsonErrorCode(body, want string) bool {
	var decoded struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	return json.Unmarshal([]byte(body), &decoded) == nil && decoded.Error.Code == want
}

func contains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
