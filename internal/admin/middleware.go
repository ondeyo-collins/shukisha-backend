package admin

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/shukisha/backend/internal/identity"
	"gorm.io/gorm"
)

type contextKey string

const AdminRoleKey contextKey = "admin_role"
const AdminIDKey contextKey = "admin_id"

const (
	RoleSuperAdmin = "SUPER_ADMIN"
	RoleLogistics  = "LOGISTICS"
	RoleTransport  = "TRANSPORT"
	RoleFinance    = "FINANCE"
	RoleSupport    = "SUPPORT"
)

func IsCanonicalAdminRole(role string) bool {
	switch role {
	case RoleSuperAdmin, RoleLogistics, RoleTransport, RoleFinance, RoleSupport:
		return true
	default:
		return false
	}
}

func IsCanonicalAdminStatus(status string) bool {
	switch status {
	case "ACTIVE", "SUSPENDED":
		return true
	default:
		return false
	}
}

func RequireAnyRole(db *gorm.DB, allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return identity.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(identity.UserIDKey).(uuid.UUID)
			if !ok {
				http.Error(w, "Unauthorized: missing user ID", http.StatusUnauthorized)
				return
			}

			var admin AdminUser
			if err := db.Where("user_id = ?", userID).First(&admin).Error; err != nil {
				http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
				return
			}

			if admin.Status != "ACTIVE" {
				http.Error(w, "Forbidden: Admin account is not active", http.StatusForbidden)
				return
			}

			if !IsCanonicalAdminRole(admin.Role) {
				http.Error(w, "Forbidden: Admin role misconfigured", http.StatusForbidden)
				return
			}

			// SUPER_ADMIN bypass
			if admin.Role != RoleSuperAdmin {
				if _, ok := allowed[admin.Role]; !ok {
					http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
					return
				}
			}

			ctx := context.WithValue(r.Context(), AdminRoleKey, admin.Role)
			ctx = context.WithValue(ctx, AdminIDKey, admin.ID)
			next(w, r.WithContext(ctx))
		})
	}
}

func AdminRequired(db *gorm.DB, next http.HandlerFunc) http.HandlerFunc {
	return RequireAnyRole(db, RoleSuperAdmin, RoleLogistics, RoleTransport, RoleFinance, RoleSupport)(next)
}

func SuperAdminRequired(db *gorm.DB, next http.HandlerFunc) http.HandlerFunc {
	return RequireAnyRole(db, RoleSuperAdmin)(next)
}
