package schedulingimpact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/banggok/sched_mind/backend/internal/shared/httpjson"
)

const ConfirmationTokenHeader = "X-Scheduling-Impact-Token"

type Mode string

const (
	ModeOrdinary   Mode = "ordinary"
	ModeActualDate Mode = "actual_date"
	ModeBulkReopen Mode = "bulk_reopen"
)

type Project struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"-"`
	Version int64  `json:"-"`
}

type Kind string

const (
	ConfirmationRequired Kind = "SCHEDULING_IMPACT_CONFIRMATION_REQUIRED"
	LockedProjectImpact  Kind = "SCHEDULING_LOCKED_PROJECT_IMPACT"
	StaleImpact          Kind = "SCHEDULING_IMPACT_STALE"
)

type Error struct {
	Kind           Kind
	Token          string
	LockedProjects []Project
	OpenProjects   []Project
}

func (value Error) Error() string {
	switch value.Kind {
	case ConfirmationRequired:
		return "scheduling impact confirmation is required"
	case LockedProjectImpact:
		return "scheduling mutation would affect a locked project"
	case StaleImpact:
		return "scheduling impact changed after preview"
	default:
		return "scheduling impact guard rejected the mutation"
	}
}

type requestContextKey struct{}
type operationContextKey struct{}

type operation struct {
	Enabled        bool
	OwnerProjectID string
	Mode           Mode
}

func CaptureToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		token := strings.TrimSpace(request.Header.Get(ConfirmationTokenHeader))
		ctx := context.WithValue(request.Context(), requestContextKey{}, token)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

func WithOperation(ctx context.Context, ownerProjectID string, mode Mode) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, operationContextKey{}, operation{
		Enabled:        true,
		OwnerProjectID: ownerProjectID,
		Mode:           mode,
	})
}

// WithPreviewOperation starts a fresh scheduling-impact preview on top of the
// caller context while deliberately ignoring any previously supplied
// confirmation token. This is used by server-side simulations (for example,
// Project Reopen fixed-point discovery) that must always observe the current
// impact set rather than accidentally accepting a client token.
func WithPreviewOperation(ctx context.Context, ownerProjectID string, mode Mode) context.Context {
	ctx = WithOperation(ctx, ownerProjectID, mode)
	return context.WithValue(ctx, requestContextKey{}, "")
}

func Operation(ctx context.Context) (enabled bool, ownerProjectID string, mode Mode, confirmationToken string) {
	if ctx == nil {
		return false, "", ModeOrdinary, ""
	}
	value, _ := ctx.Value(operationContextKey{}).(operation)
	token, _ := ctx.Value(requestContextKey{}).(string)
	if value.Mode == "" {
		value.Mode = ModeOrdinary
	}
	return value.Enabled, value.OwnerProjectID, value.Mode, token
}

func Guard(ctx context.Context, signature string, lockedProjects, openProjects []Project) error {
	enabled, ownerProjectID, mode, suppliedToken := Operation(ctx)
	if !enabled || mode == ModeBulkReopen {
		return nil
	}
	lockedProjects = normalizeProjects(lockedProjects, ownerProjectID)
	openProjects = normalizeProjects(openProjects, ownerProjectID)
	if len(lockedProjects) == 0 && len(openProjects) == 0 {
		return nil
	}
	token := impactToken(ownerProjectID, mode, signature, lockedProjects, openProjects)
	if mode != ModeActualDate && len(lockedProjects) > 0 {
		return Error{Kind: LockedProjectImpact, Token: token, LockedProjects: lockedProjects, OpenProjects: openProjects}
	}
	if suppliedToken == "" {
		return Error{Kind: ConfirmationRequired, Token: token, LockedProjects: lockedProjects, OpenProjects: openProjects}
	}
	if suppliedToken != token {
		return Error{Kind: StaleImpact, Token: token, LockedProjects: lockedProjects, OpenProjects: openProjects}
	}
	return nil
}

func normalizeProjects(values []Project, ownerProjectID string) []Project {
	byID := make(map[string]Project, len(values))
	for _, value := range values {
		if value.ID == "" || value.ID == ownerProjectID {
			continue
		}
		byID[value.ID] = value
	}
	out := make([]Project, 0, len(byID))
	for _, value := range byID {
		out = append(out, value)
	}
	sort.Slice(out, func(left, right int) bool {
		if strings.EqualFold(out[left].Name, out[right].Name) {
			return out[left].ID < out[right].ID
		}
		return strings.ToLower(out[left].Name) < strings.ToLower(out[right].Name)
	})
	return out
}

func impactToken(ownerProjectID string, mode Mode, signature string, lockedProjects, openProjects []Project) string {
	parts := []string{ownerProjectID, string(mode), signature}
	for _, value := range lockedProjects {
		parts = append(parts, "locked:"+value.ID+":"+integer(value.Version))
	}
	for _, value := range openProjects {
		parts = append(parts, "open:"+value.ID+":"+integer(value.Version))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(digest[:])
}

func integer(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	buffer := [20]byte{}
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		buffer[index] = '-'
	}
	return string(buffer[index:])
}

type errorResponse struct {
	Code    Kind         `json:"code"`
	Message string       `json:"message"`
	Details impactDetail `json:"details"`
}

type impactDetail struct {
	Token          string    `json:"token"`
	LockedProjects []Project `json:"lockedProjects"`
	OpenProjects   []Project `json:"openProjects"`
}

func WriteHTTPError(response http.ResponseWriter, err error) bool {
	var impactError Error
	if !errors.As(err, &impactError) {
		return false
	}
	message := impactError.Error()
	httpjson.Write(response, http.StatusConflict, errorResponse{
		Code:    impactError.Kind,
		Message: message,
		Details: impactDetail{
			Token:          impactError.Token,
			LockedProjects: impactError.LockedProjects,
			OpenProjects:   impactError.OpenProjects,
		},
	})
	return true
}
