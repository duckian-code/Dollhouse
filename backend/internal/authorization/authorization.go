// Package authorization derives principals and roles exclusively from the
// verified JWT claims supplied by API Gateway.
package authorization

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/dollhouse-app/dollhouse/backend/pkg/response"
)

const (
	UserGroup  = "User"
	AdminGroup = "Admin"
)

// Principal is the authenticated caller represented by verified Cognito
// claims. Subject is the only identifier handlers may use for owned objects.
type Principal struct {
	Subject string
	Groups  map[string]struct{}
}

// Authenticate returns the caller represented by API Gateway's verified JWT.
func Authenticate(request events.APIGatewayV2HTTPRequest) (Principal, *events.APIGatewayV2HTTPResponse) {
	if request.RequestContext.Authorizer == nil || request.RequestContext.Authorizer.JWT == nil {
		return Principal{}, unauthenticated()
	}
	claims := request.RequestContext.Authorizer.JWT.Claims
	subject := strings.TrimSpace(claims["sub"])
	if subject == "" {
		return Principal{}, unauthenticated()
	}
	return Principal{
		Subject: subject, Groups: parseGroups(claims["cognito:groups"]),
	}, nil
}

// RequireGroup authenticates the caller and enforces a Cognito group claim.
func RequireGroup(request events.APIGatewayV2HTTPRequest, group string) (Principal, *events.APIGatewayV2HTTPResponse) {
	principal, failure := Authenticate(request)
	if failure != nil {
		return Principal{}, failure
	}
	if !principal.InGroup(group) {
		denied := response.Error(http.StatusForbidden, "forbidden", "insufficient permissions")
		return Principal{}, &denied
	}
	return principal, nil
}

// RequireAdmin gates administrator-only routes and catalog mutations.
func RequireAdmin(request events.APIGatewayV2HTTPRequest) (Principal, *events.APIGatewayV2HTTPResponse) {
	return RequireGroup(request, AdminGroup)
}

// InGroup reports whether Cognito assigned the principal to group.
func (p Principal) InGroup(group string) bool {
	_, ok := p.Groups[group]
	return ok
}

func unauthenticated() *events.APIGatewayV2HTTPResponse {
	failure := response.Error(http.StatusUnauthorized, "unauthenticated", "valid Cognito authentication is required")
	return &failure
}

func parseGroups(raw string) map[string]struct{} {
	groups := make(map[string]struct{})
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return groups
	}
	var values []string
	if json.Unmarshal([]byte(raw), &values) != nil {
		trimmed := strings.Trim(raw, "[]")
		values = strings.FieldsFunc(trimmed, func(r rune) bool { return r == ',' || r == ' ' })
	}
	for _, value := range values {
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if value != "" {
			groups[value] = struct{}{}
		}
	}
	return groups
}
