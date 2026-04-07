package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/shukisha/backend/internal/admin"
	"github.com/shukisha/backend/internal/drivers"
	"github.com/shukisha/backend/internal/feedback"
	"github.com/shukisha/backend/internal/identity"
	"github.com/shukisha/backend/internal/ledger"
	"github.com/shukisha/backend/internal/parcels"
	"github.com/shukisha/backend/internal/platform/cache"
	"github.com/shukisha/backend/internal/platform/config"
	"github.com/shukisha/backend/internal/platform/db"
	"github.com/shukisha/backend/internal/platform/middleware"
	"github.com/shukisha/backend/internal/platform/sms"
	"github.com/shukisha/backend/internal/platform/swagger"
	"github.com/shukisha/backend/internal/routes"
	"github.com/shukisha/backend/internal/stopalerts"
)

func main() {
	// Initialize Database
	if _, err := db.Init(); err != nil {
		log.Fatalf("Critical: Could not initialize database: %v", err)
	}

	// Auto-Migrate Schema
	fmt.Println("Running database migrations...")
	err := db.DB.AutoMigrate(
		&identity.User{},
		&identity.OTPChallenge{},
		&routes.Route{},
		&routes.Stop{},
		&routes.RouteStop{},
		&drivers.DriverProfile{},
		&drivers.DriverSession{},
		&stopalerts.StopAlert{},
		&parcels.Parcel{},
		&parcels.ParcelTrackingEvent{},
		&parcels.Payment{},
		&parcels.ParcelNote{},
		&parcels.ParcelClaim{},
		&admin.AdminUser{},
		&admin.AdminCredential{},
		&admin.AuditEvent{},
		&ledger.LedgerEntry{},
		&ledger.PayoutBatch{},
		&feedback.Rating{},
		&feedback.Report{},
	)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Initialize Cache
	if _, err := cache.Init(); err != nil {
		log.Fatalf("Critical: Could not initialize cache: %v", err)
	}

	// Initialize OTP/SMS Provider
	smsProvider := sms.NewTwilioProvider()

	// Initialize Identity Module
	idService := identity.NewIdentityService(db.DB, smsProvider)
	idHandler := identity.NewAuthHandler(idService)

	// Initialize Routes Module
	routeService := routes.NewRouteService(db.DB)
	routeHandler := routes.NewRouteHandler(routeService)

	// Initialize Driver Module
	driverService := drivers.NewDriverService(db.DB, cache.RDB)
	driverHandler := drivers.NewDriverHandler(driverService)

	// Initialize Stop Alert Module
	stopAlertService := stopalerts.NewStopAlertService(db.DB)
	stopAlertHandler := stopalerts.NewStopAlertHandler(stopAlertService)

	// Initialize Parcel Module
	parcelService := parcels.NewParcelService(db.DB)
	parcelHandler := parcels.NewParcelHandler(parcelService)

	// Initialize Admin Module
	adminSvc := admin.NewAdminService(db.DB)
	adminHandler := admin.NewAdminHandler(adminSvc)
	adminAuthSvc := admin.NewAdminAuthService(db.DB)
	adminAuthHandler := admin.NewAdminAuthHandler(adminAuthSvc)

	feedbackSvc := feedback.NewFeedbackService(db.DB)
	feedbackHandler := feedback.NewFeedbackHandler(feedbackSvc)

	port := config.GetEnv("PORT", "8080")
	if port[0] != ':' {
		port = ":" + port
	}
	fmt.Printf("Shukisha API starting on %s...\n", port)

	// Auth Routes
	http.HandleFunc("/v1/auth/otp/request", idHandler.RequestOTP)
	http.HandleFunc("/v1/auth/otp/verify", idHandler.VerifyOTP)
	http.HandleFunc("/v1/auth/refresh", idHandler.RefreshToken)
	http.HandleFunc("/v1/auth/admin/login", adminAuthHandler.Login) // Email+Password for admin UI

	// Route Routes
	http.HandleFunc("/v1/routes/search", routeHandler.SearchRoutes)
	http.HandleFunc("/v1/routes/", routeHandler.GetRouteDetails)
	http.HandleFunc("/v1/stops/nearest", routeHandler.SearchNearestStops)

	// Driver Routes (Protected)
	http.HandleFunc("/v1/driver/onboarding", identity.AuthMiddleware(driverHandler.Onboard))
	http.HandleFunc("/v1/driver/session/start", identity.AuthMiddleware(driverHandler.StartSession))
	http.HandleFunc("/v1/driver/session/stop", identity.AuthMiddleware(driverHandler.StopSession))
	http.HandleFunc("/v1/driver/session/heartbeat", identity.AuthMiddleware(driverHandler.Heartbeat))

	http.HandleFunc("/v1/driver/stop-requests", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			identity.AuthMiddleware(stopAlertHandler.GetDriverAlerts)(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	http.HandleFunc("/v1/driver/stop-requests/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/ack") {
			identity.AuthMiddleware(stopAlertHandler.AckAlert)(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	http.HandleFunc("/v1/driver/parcels/offers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			identity.AuthMiddleware(parcelHandler.GetDriverOffers)(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	http.HandleFunc("/v1/driver/parcels/", func(w http.ResponseWriter, r *http.Request) {
		// All other driver parcel actions /v1/driver/parcels/{id}/{action}
		identity.AuthMiddleware(parcelHandler.DriverParcelAction)(w, r)
	})

	// Stop Alert Routes (Protected)
	http.HandleFunc("/v1/stop-alerts", identity.AuthMiddleware(stopAlertHandler.CreateAlert))
	http.HandleFunc("/v1/stop-alerts/active", identity.AuthMiddleware(stopAlertHandler.GetActiveAlerts))
	// Internal/Test Route for proximity check
	http.HandleFunc("/v1/stop-alerts/check-proximity", stopAlertHandler.TriggerProximityCheck)

	// Parcel Routes
	http.HandleFunc("/v1/parcels", identity.AuthMiddleware(parcelHandler.BookParcel))
	http.HandleFunc("/v1/payments/callback", parcelHandler.PaymentCallback) // M-Pesa Callback
	http.HandleFunc("/v1/parcels/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/pay") {
			identity.AuthMiddleware(parcelHandler.PayParcel)(w, r)
		} else {
			parcelHandler.GetParcelStatus(w, r)
		}
	})

	// Swagger & Documentation
	http.HandleFunc("/v1/openapi.yaml", swagger.SpecHandler("docs/openapi.yaml"))
	http.HandleFunc("/swagger", swagger.SwaggerHandler)

	// Admin Routes (Protected by Admin Middleware)
	http.HandleFunc("/v1/admin/stats", admin.RequireAnyRole(db.DB, admin.RoleSuperAdmin, admin.RoleLogistics, admin.RoleTransport, admin.RoleFinance, admin.RoleSupport)(adminHandler.GetDashboardStats))

	http.HandleFunc("/v1/admin/drivers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			admin.RequireAnyRole(db.DB, admin.RoleTransport)(adminHandler.GetDashboardStats)(w, r) // Fallback placeholder
		} else {
			admin.RequireAnyRole(db.DB, admin.RoleTransport)(adminHandler.UpdateDriverStatus)(w, r)
		}
	})
	http.HandleFunc("/v1/admin/drivers/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/ping") {
			admin.RequireAnyRole(db.DB, admin.RoleTransport)(adminHandler.PingDriver)(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	http.HandleFunc("/v1/admin/parcels", admin.RequireAnyRole(db.DB, admin.RoleLogistics)(adminHandler.GetParcels))
	http.HandleFunc("/v1/admin/parcels/exceptions", admin.RequireAnyRole(db.DB, admin.RoleLogistics)(adminHandler.GetParcelExceptions))
	http.HandleFunc("/v1/admin/parcels/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/override") {
			admin.RequireAnyRole(db.DB, admin.RoleLogistics)(adminHandler.OverrideParcel)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/events") {
			admin.RequireAnyRole(db.DB, admin.RoleLogistics)(adminHandler.GetParcelEvents)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/claims") {
			admin.RequireAnyRole(db.DB, admin.RoleLogistics)(adminHandler.ParcelClaims)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/notes") {
			admin.RequireAnyRole(db.DB, admin.RoleLogistics)(adminHandler.AddParcelNote)(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/notify-driver") {
			admin.RequireAnyRole(db.DB, admin.RoleLogistics)(adminHandler.NotifyDriver)(w, r)
		} else if r.Method == http.MethodGet {
			admin.RequireAnyRole(db.DB, admin.RoleLogistics)(adminHandler.GetParcelDetail)(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	http.HandleFunc("/v1/admin/claims", admin.RequireAnyRole(db.DB, admin.RoleLogistics)(adminHandler.ListClaims))
	http.HandleFunc("/v1/admin/claims/", admin.RequireAnyRole(db.DB, admin.RoleLogistics)(adminHandler.UpdateClaim))

	http.HandleFunc("/v1/admin/fleet/live", admin.RequireAnyRole(db.DB, admin.RoleTransport)(adminHandler.GetLiveFleet))
	http.HandleFunc("/v1/admin/routes", admin.RequireAnyRole(db.DB, admin.RoleTransport)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			adminHandler.CreateRoute(w, r)
		} else {
			adminHandler.GetRoutes(w, r)
		}
	}))
	http.HandleFunc("/v1/admin/stops", admin.RequireAnyRole(db.DB, admin.RoleTransport)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			adminHandler.CreateStop(w, r)
		} else {
			adminHandler.GetStops(w, r)
		}
	}))

	http.HandleFunc("/v1/admin/ledger/unpaid", admin.RequireAnyRole(db.DB, admin.RoleFinance)(adminHandler.GetUnpaidLedger))
	http.HandleFunc("/v1/admin/payouts/execute", admin.RequireAnyRole(db.DB, admin.RoleFinance)(adminHandler.ExecutePayouts))

	http.HandleFunc("/v1/admin/users/search", admin.RequireAnyRole(db.DB, admin.RoleSupport)(adminHandler.SearchUsers))
	http.HandleFunc("/v1/admin/users/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/rescue-auth") {
			admin.RequireAnyRole(db.DB, admin.RoleSupport)(adminHandler.RescueAuth)(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// Super Admin Overseer Route
	http.HandleFunc("/v1/admin/audit-logs", admin.RequireAnyRole(db.DB, admin.RoleSuperAdmin)(adminHandler.GetAuditLogs))

	// Administration Management (Super Admin Only)
	http.HandleFunc("/v1/admin/management", admin.RequireAnyRole(db.DB, admin.RoleSuperAdmin)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			adminHandler.CreateAdmin(w, r)
		} else {
			adminHandler.ListAdmins(w, r)
		}
	}))
	http.HandleFunc("/v1/admin/management/", admin.RequireAnyRole(db.DB, admin.RoleSuperAdmin)(adminHandler.UpdateAdmin))

	// Feedback Routes
	http.HandleFunc("/v1/ratings", identity.AuthMiddleware(feedbackHandler.SubmitRating))
	http.HandleFunc("/v1/reports", identity.AuthMiddleware(feedbackHandler.SubmitReport))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})

	if err := http.ListenAndServe(port, middleware.CorsMiddleware(http.DefaultServeMux)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
