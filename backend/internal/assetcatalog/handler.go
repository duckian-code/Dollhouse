package assetcatalog

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/dollhouse-app/dollhouse/backend/pkg/response"
)

type catalogService interface {
	GetCatalog(context.Context) (Catalog, error)
}

// Handler serves the authenticated asset catalog route.
type Handler struct {
	service catalogService
}

// NewHandler creates an asset catalog API handler.
func NewHandler(service catalogService) *Handler { return &Handler{service: service} }

// GetCatalog returns approved assets and their short-lived retrieval URLs.
func (h *Handler) GetCatalog(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if !hasVerifiedIdentity(request) {
		return response.Error(http.StatusUnauthorized, "unauthenticated", "valid Cognito authentication is required"), nil
	}
	catalog, err := h.service.GetCatalog(ctx)
	if err != nil {
		log.Printf("asset catalog request failed error=%q", err)
		return response.Error(http.StatusInternalServerError, "internal_error", "an internal error occurred"), nil
	}
	return response.JSON(http.StatusOK, map[string]any{"data": catalog})
}

func hasVerifiedIdentity(request events.APIGatewayV2HTTPRequest) bool {
	return request.RequestContext.Authorizer != nil &&
		request.RequestContext.Authorizer.JWT != nil &&
		strings.TrimSpace(request.RequestContext.Authorizer.JWT.Claims["sub"]) != ""
}
