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

func AdminRequired(db *gorm.DB, next http.HandlerFunc) http.HandlerFunc {
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

		ctx := context.WithValue(r.Context(), AdminRoleKey, admin.Role)
		next(w, r.WithContext(ctx))
	})
}

func SuperAdminRequired(db *gorm.DB, next http.HandlerFunc) http.HandlerFunc {
	return AdminRequired(db, func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(AdminRoleKey).(string)
		if !ok || role != "SUPER_ADMIN" {
			http.Error(w, "Forbidden: Superadmin access required", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}
