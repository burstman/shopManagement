package app

import (
	"net/http"
	"shopDashboard/app/db"
	"shopDashboard/app/handlers"

	"github.com/anthdm/superkit/kit"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func InitializeMiddleware(router *chi.Mux) {
	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.RealIP)
}

func checkSetup(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		skip := r.URL.Path == "/setup" || r.URL.Path == "/login" || r.URL.Path == "/healthz" ||
			len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" ||
			len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/public/"
		if skip {
			next.ServeHTTP(w, r)
			return
		}
		has, err := db.HasAnySuperAdmin()
		if err != nil || !has {
			http.Redirect(w, r, "/setup", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func InitializeRoutes(router *chi.Mux) {
	router.Use(checkSetup)

	authCfg := kit.AuthenticationConfig{
		AuthFunc:    handlers.LoadAuth,
		RedirectURL: "/login",
	}

	router.Group(func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, rq *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		r.Get("/setup", kit.Handler(handlers.HandleSetup))
		r.Post("/setup", kit.Handler(handlers.HandleSetup))
		r.Get("/login", kit.Handler(handlers.HandleLogin))
		r.Post("/login", kit.Handler(handlers.HandleLogin))
		r.Post("/logout", kit.Handler(handlers.HandleLogout))

		r.Post("/api/error", kit.Handler(handlers.HandleReportError))
		r.Post("/api/errors", kit.Handler(handlers.HandleReportError))
		r.Post("/api/warn", kit.Handler(handlers.HandleReportWarn))

		r.Handle("/public/*", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
	})

	router.Group(func(r chi.Router) {
		r.Use(kit.WithAuthentication(authCfg, true))

		r.Get("/", kit.Handler(handlers.HandleServersList))
		r.Get("/admins", kit.Handler(handlers.HandleAdmins))
		r.Post("/admins", kit.Handler(handlers.HandleAdmins))
		r.Get("/affiliate/{id}", kit.Handler(handlers.HandleAffiliateDashboard))
		r.Put("/affiliate/{id}/shop-url", kit.Handler(handlers.HandleUpdateDomain))
		r.Post("/affiliate/{id}/reset-credentials", kit.Handler(handlers.HandleResetCredentials))
		r.Put("/affiliate/{id}/dashboard-url", kit.Handler(handlers.HandleUpdateDashboardURL))
		r.Get("/affiliate/{id}/ping", kit.Handler(handlers.HandlePing))
	})
}
