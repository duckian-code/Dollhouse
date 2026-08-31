package profiledoll

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
	"github.com/dollhouse-app/dollhouse/backend/internal/assetcatalog"
	"github.com/dollhouse-app/dollhouse/backend/internal/authorization"
	"github.com/dollhouse-app/dollhouse/backend/internal/observability"
	"github.com/dollhouse-app/dollhouse/backend/pkg/response"
)

const maxRequestBodyBytes = 32 * 1024

// Handlers implements the four profile and doll Lambda routes.
type Handlers struct {
	store          Store
	assetValidator interface {
		Validate(context.Context, []assetcatalog.Reference) error
	}
	now func() time.Time
}

// NewHandlers creates handlers with production clock behavior.
func NewHandlers(store Store, assetValidator interface {
	Validate(context.Context, []assetcatalog.Reference) error
}) *Handlers {
	return &Handlers{store: store, assetValidator: assetValidator, now: time.Now}
}

// GetProfile returns the signed-in user's profile, creating its initial record
// from verified Cognito claims on first access.
func (h *Handlers) GetProfile(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	identity, failure := identityFrom(request)
	if failure != nil {
		return *failure, nil
	}
	profile, err := h.store.EnsureUser(ctx, identity, h.timestamp())
	if err != nil {
		return internalError(ctx, "get profile", err), nil
	}
	return response.JSON(http.StatusOK, map[string]any{"data": map[string]any{"profile": profile}})
}

type updateProfileRequest struct {
	Username    optionalJSON `json:"-"`
	DisplayName optionalJSON `json:"-"`
	Bio         optionalJSON `json:"-"`
}

type optionalJSON struct {
	Set   bool
	Value *string
}

func (r *updateProfileRequest) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for name, raw := range fields {
		var target *optionalJSON
		switch name {
		case "username":
			target = &r.Username
		case "displayName":
			target = &r.DisplayName
		case "bio":
			target = &r.Bio
		default:
			return fmt.Errorf("unknown field %q", name)
		}
		target.Set = true
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			target.Value = nil
			continue
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("%s must be a string or null", name)
		}
		target.Value = &value
	}
	return nil
}

// UpdateProfile applies only fields explicitly included in the request body.
func (h *Handlers) UpdateProfile(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	identity, failure := identityFrom(request)
	if failure != nil {
		return *failure, nil
	}
	var input updateProfileRequest
	if failure = decodeBody(request, &input); failure != nil {
		return *failure, nil
	}
	changes, err := validateProfileUpdate(input)
	if err != nil {
		return response.Error(http.StatusBadRequest, "validation_failed", err.Error()), nil
	}
	if _, err = h.store.EnsureUser(ctx, identity, h.timestamp()); err != nil {
		return internalError(ctx, "ensure profile", err), nil
	}
	profile, err := h.store.UpdateProfile(ctx, identity.UserID, changes, h.timestamp())
	if err != nil {
		return h.failure(ctx, "update profile", err), nil
	}
	return response.JSON(http.StatusOK, map[string]any{"data": map[string]any{"profile": profile}})
}

// UsernameAvailability reports whether the normalized username is unreserved.
// This check is advisory; UpdateProfile performs the authoritative claim.
func (h *Handlers) UsernameAvailability(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	identity, failure := identityFrom(request)
	if failure != nil {
		return *failure, nil
	}
	username, err := validateUsername(request.QueryStringParameters["username"])
	if err != nil {
		return response.Error(http.StatusBadRequest, "validation_failed", err.Error()), nil
	}
	available, err := h.store.UsernameAvailable(ctx, NormalizeUsername(username), identity.UserID)
	if err != nil {
		return internalError(ctx, "check username availability", err), nil
	}
	return response.JSON(http.StatusOK, map[string]any{"data": map[string]any{
		"username": username, "available": available,
	}})
}

type updateDollRequest struct {
	BodyAssetID      string   `json:"bodyAssetId"`
	HairAssetID      string   `json:"hairAssetId"`
	EyesAssetID      string   `json:"eyesAssetId"`
	NoseAssetID      string   `json:"noseAssetId"`
	MouthAssetID     string   `json:"mouthAssetId"`
	ClothingAssetIDs []string `json:"clothingAssetIds"`
}

// GetDoll returns the saved doll configuration.
func (h *Handlers) GetDoll(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	identity, failure := identityFrom(request)
	if failure != nil {
		return *failure, nil
	}
	if _, err := h.store.EnsureUser(ctx, identity, h.timestamp()); err != nil {
		return internalError(ctx, "ensure profile", err), nil
	}
	configuration, err := h.store.GetDoll(ctx, identity.UserID)
	if err != nil {
		return internalError(ctx, "get doll", err), nil
	}
	if configuration == nil {
		return response.Error(http.StatusNotFound, "not_found", "doll configuration not found"), nil
	}
	return response.JSON(http.StatusOK, map[string]any{"data": map[string]any{"configuration": configuration}})
}

// UpdateDoll validates and replaces the signed-in user's full configuration.
func (h *Handlers) UpdateDoll(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	identity, failure := identityFrom(request)
	if failure != nil {
		return *failure, nil
	}
	var input updateDollRequest
	if failure = decodeBody(request, &input); failure != nil {
		return *failure, nil
	}
	configuration, err := validateDoll(input)
	if err != nil {
		return response.Error(http.StatusBadRequest, "validation_failed", err.Error()), nil
	}
	if err = h.assetValidator.Validate(ctx, assetReferences(configuration)); err != nil {
		if assetcatalog.IsSelectionError(err) {
			return response.Error(http.StatusBadRequest, "validation_failed", err.Error()), nil
		}
		return internalError(ctx, "validate doll assets", err), nil
	}
	now := h.timestamp()
	if _, err = h.store.EnsureUser(ctx, identity, now); err != nil {
		return internalError(ctx, "ensure profile", err), nil
	}
	configuration, err = h.store.UpdateDoll(ctx, identity.UserID, configuration, now)
	if err != nil {
		return internalError(ctx, "update doll", err), nil
	}
	return response.JSON(http.StatusOK, map[string]any{"data": map[string]any{"configuration": configuration}})
}

func identityFrom(request events.APIGatewayV2HTTPRequest) (Identity, *events.APIGatewayV2HTTPResponse) {
	principal, failure := authorization.Authenticate(request)
	if failure != nil {
		return Identity{}, failure
	}
	return Identity{UserID: principal.Subject}, nil
}

func decodeBody(request events.APIGatewayV2HTTPRequest, target any) *events.APIGatewayV2HTTPResponse {
	body := []byte(request.Body)
	if request.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			failure := response.Error(http.StatusBadRequest, "invalid_request", "request body is not valid base64")
			return &failure
		}
		body = decoded
	}
	if len(body) == 0 {
		failure := response.Error(http.StatusBadRequest, "invalid_request", "request body is required")
		return &failure
	}
	if len(body) > maxRequestBodyBytes {
		failure := response.Error(http.StatusBadRequest, "invalid_request", "request body is too large")
		return &failure
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		failure := response.Error(http.StatusBadRequest, "invalid_request", "request body must be valid JSON: "+err.Error())
		return &failure
	}
	if err := ensureEOF(decoder); err != nil {
		failure := response.Error(http.StatusBadRequest, "invalid_request", "request body must contain one JSON object")
		return &failure
	}
	return nil
}

func ensureEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON value")
	}
	return nil
}

func validateProfileUpdate(input updateProfileRequest) (ProfileChanges, error) {
	if !input.Username.Set && !input.DisplayName.Set && !input.Bio.Set {
		return ProfileChanges{}, errors.New("at least one profile field is required")
	}
	changes := ProfileChanges{
		Username:    OptionalString{Set: input.Username.Set, Value: input.Username.Value},
		DisplayName: OptionalString{Set: input.DisplayName.Set, Value: input.DisplayName.Value},
		Bio:         OptionalString{Set: input.Bio.Set, Value: input.Bio.Value},
	}
	if changes.Username.Set {
		if changes.Username.Value == nil {
			return ProfileChanges{}, errors.New("username must be a string")
		}
		value, err := validateUsername(*changes.Username.Value)
		if err != nil {
			return ProfileChanges{}, err
		}
		changes.Username.Value = &value
	}
	if changes.DisplayName.Set {
		if changes.DisplayName.Value == nil || strings.TrimSpace(*changes.DisplayName.Value) == "" {
			return ProfileChanges{}, errors.New("displayName must be a non-empty string")
		}
		value := strings.TrimSpace(*changes.DisplayName.Value)
		if utf8.RuneCountInString(value) > 100 {
			return ProfileChanges{}, errors.New("displayName must be 100 characters or fewer")
		}
		changes.DisplayName.Value = &value
	}
	if changes.Bio.Set && changes.Bio.Value != nil && utf8.RuneCountInString(*changes.Bio.Value) > 500 {
		return ProfileChanges{}, errors.New("bio must be 500 characters or fewer")
	}
	return changes, nil
}

func validateDoll(input updateDollRequest) (DollConfiguration, error) {
	configuration := DollConfiguration{
		BodyAssetID: input.BodyAssetID, HairAssetID: input.HairAssetID,
		EyesAssetID: input.EyesAssetID, NoseAssetID: input.NoseAssetID,
		MouthAssetID: input.MouthAssetID, ClothingAssetIDs: input.ClothingAssetIDs,
	}
	required := []struct{ name, value string }{
		{"bodyAssetId", configuration.BodyAssetID}, {"hairAssetId", configuration.HairAssetID},
		{"eyesAssetId", configuration.EyesAssetID}, {"noseAssetId", configuration.NoseAssetID},
		{"mouthAssetId", configuration.MouthAssetID},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return DollConfiguration{}, fmt.Errorf("%s is required", field.name)
		}
	}
	if configuration.ClothingAssetIDs == nil {
		return DollConfiguration{}, errors.New("clothingAssetIds is required")
	}
	seen := make(map[string]struct{}, len(configuration.ClothingAssetIDs))
	for _, id := range configuration.ClothingAssetIDs {
		if strings.TrimSpace(id) == "" {
			return DollConfiguration{}, errors.New("clothingAssetIds cannot contain an empty ID")
		}
		if _, exists := seen[id]; exists {
			return DollConfiguration{}, fmt.Errorf("clothingAssetIds contains duplicate ID %q", id)
		}
		seen[id] = struct{}{}
	}
	return configuration, nil
}

func assetReferences(configuration DollConfiguration) []assetcatalog.Reference {
	references := []assetcatalog.Reference{
		{Field: "bodyAssetId", AssetID: configuration.BodyAssetID, Category: "body"},
		{Field: "hairAssetId", AssetID: configuration.HairAssetID, Category: "hair"},
		{Field: "eyesAssetId", AssetID: configuration.EyesAssetID, Category: "eyes"},
		{Field: "noseAssetId", AssetID: configuration.NoseAssetID, Category: "nose"},
		{Field: "mouthAssetId", AssetID: configuration.MouthAssetID, Category: "mouth"},
	}
	for index, assetID := range configuration.ClothingAssetIDs {
		references = append(references, assetcatalog.Reference{
			Field: fmt.Sprintf("clothingAssetIds[%d]", index), AssetID: assetID, Category: "clothing",
		})
	}
	return references
}

func (h *Handlers) timestamp() string {
	return h.now().UTC().Truncate(time.Second).Format(time.RFC3339)
}

func internalError(ctx context.Context, operation string, err error) events.APIGatewayV2HTTPResponse {
	observability.Logger(ctx).ErrorContext(ctx, "profile/doll request failed", "operation", operation, "error", err)
	return response.Error(http.StatusInternalServerError, "internal_error", "an internal error occurred")
}

func (h *Handlers) failure(ctx context.Context, operation string, err error) events.APIGatewayV2HTTPResponse {
	var domain *DomainError
	if errors.As(err, &domain) {
		return response.Error(domain.Status, domain.Code, domain.Message)
	}
	return internalError(ctx, operation, err)
}

func validateUsername(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("username must be a non-empty string")
	}
	if utf8.RuneCountInString(value) > 50 {
		return "", errors.New("username must be 50 characters or fewer")
	}
	return value, nil
}
