package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"fast-bypass/internal/app"
	"fast-bypass/internal/auth"
)

func New(a *app.App) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   a.Cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		dbOK := a.Store.Ping(ctx) == nil
		mtOK := a.MT.Ping() == nil
		status := "ok"
		code := http.StatusOK
		if !dbOK {
			status = "degraded"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(`{"status":"` + status + `","db":` + boolStr(dbOK) + `,"mikrotik":` + boolStr(mtOK) + `}`))
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/auth/login", a.HandleLogin)
		api.Post("/auth/refresh", a.HandleRefresh)

		api.Group(func(pr chi.Router) {
			pr.Use(auth.Middleware(a.Auth))
			pr.Post("/auth/logout", a.HandleLogout)
			pr.Get("/me", a.HandleMe)
			pr.Patch("/me", a.HandlePatchMe)
			pr.Post("/me/password", a.HandleChangePassword)
			pr.Get("/me/quota", a.HandleMeQuota)

			pr.Get("/vpn-users", a.HandleListVPNUsers)
			pr.Post("/vpn-users", a.HandleCreateVPNUser)
			pr.Get("/vpn-users/by-name/{name}", a.HandleGetVPNUserByName)
			pr.Patch("/vpn-users/by-name/{name}", a.HandlePatchVPNUserByName)
			pr.Delete("/vpn-users/by-name/{name}", a.HandleDeleteVPNUserByName)
			pr.Post("/vpn-users/by-name/{name}/assign-profile", a.HandleAssignProfileByName)
			pr.Get("/vpn-users/by-name/{name}/connection", a.HandleConnectionBundleByName)
			pr.Get("/vpn-users/by-name/{name}/ovpn", a.HandleDownloadOvpnByName)
			pr.Delete("/vpn-users/by-name/{name}/profiles/{profileRowId}", a.HandleRemoveProfileByName)
			pr.Get("/vpn-users/{id}", a.HandleGetVPNUser)
			pr.Patch("/vpn-users/{id}", a.HandlePatchVPNUser)
			pr.Delete("/vpn-users/{id}", a.HandleDeleteVPNUser)
			pr.Post("/vpn-users/{id}/assign-profile", a.HandleAssignProfile)
			pr.Get("/vpn-users/{id}/connection", a.HandleConnectionBundle)
			pr.Get("/vpn-users/{id}/ovpn", a.HandleDownloadOvpn)
			pr.Delete("/vpn-users/{id}/profiles/{profileRowId}", a.HandleRemoveProfile)

			pr.Get("/renewals", a.HandleManagerRenewals)

			pr.Route("/admin", func(ar chi.Router) {
				ar.Use(auth.RequireAdmin)
				ar.Get("/stats", a.HandleAdminStats)
				ar.Get("/managers", a.HandleListManagers)
				ar.Post("/managers", a.HandleCreateManager)
				ar.Patch("/managers/{id}", a.HandlePatchManager)
				ar.Get("/vpn-users", a.HandleAdminListVPNUsers)
				ar.Post("/vpn-users", a.HandleAdminCreateVPNUser)
				ar.Get("/vpn-users/by-name/{name}", a.HandleAdminGetVPNUserByName)
				ar.Patch("/vpn-users/by-name/{name}", a.HandleAdminPatchVPNUserByName)
				ar.Delete("/vpn-users/by-name/{name}", a.HandleAdminDeleteVPNUserByName)
				ar.Post("/vpn-users/by-name/{name}/assign-profile", a.HandleAdminAssignProfileByName)
				ar.Get("/vpn-users/by-name/{name}/connection", a.HandleAdminConnectionBundleByName)
				ar.Get("/vpn-users/by-name/{name}/ovpn", a.HandleAdminDownloadOvpnByName)
				ar.Delete("/vpn-users/by-name/{name}/profiles/{profileRowId}", a.HandleAdminRemoveProfileByName)
				ar.Get("/vpn-users/{id}", a.HandleAdminGetVPNUser)
				ar.Patch("/vpn-users/{id}", a.HandleAdminPatchVPNUser)
				ar.Delete("/vpn-users/{id}", a.HandleAdminDeleteVPNUser)
				ar.Post("/vpn-users/{id}/assign-profile", a.HandleAdminAssignProfile)
				ar.Get("/vpn-users/{id}/connection", a.HandleAdminConnectionBundle)
				ar.Get("/vpn-users/{id}/ovpn", a.HandleAdminDownloadOvpn)
				ar.Delete("/vpn-users/{id}/profiles/{profileRowId}", a.HandleAdminRemoveProfile)
				ar.Get("/renewals", a.HandleAdminRenewals)
				ar.Post("/renewals/settle-through", a.HandleSettleThrough)
			})
		})
	})
	return r
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
