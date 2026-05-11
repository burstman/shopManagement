package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"shopDashboard/app/db"
	"shopDashboard/app/services"
	"shopDashboard/app/views/dashboard"
	"shopDashboard/app/views/layouts"

	"github.com/anthdm/superkit/kit"
	"github.com/go-chi/chi/v5"
)

func HandlePing(kit *kit.Kit) error {
	idStr := chi.URLParam(kit.Request, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return kit.Render(dashboard.PingDisplay(id, "error", 0))
	}

	aff, err := db.GetAffiliate(id)
	if err != nil || aff.ShopURL == "" {
		return kit.Render(dashboard.PingDisplay(id, "unreachable", 0))
	}

	client := &http.Client{Timeout: 5 * time.Second}
	start := time.Now()
	resp, err := client.Get(aff.ShopURL + "/health")
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return kit.Render(dashboard.PingDisplay(id, "unreachable", elapsed))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return kit.Render(dashboard.PingDisplay(id, "unhealthy", elapsed))
	}

	return kit.Render(dashboard.PingDisplay(id, "healthy", elapsed))
}

func HandleServersList(kit *kit.Kit) error {
	affiliates, err := db.GetAffiliates()
	if err != nil {
		return kit.Render(layouts.Base("Shop Servers", dashboard.ServersList(dashboard.ServersPageData{
			Error: err.Error(),
		})))
	}
	return kit.Render(layouts.Base("Shop Servers", dashboard.ServersList(dashboard.ServersPageData{
		Servers: affiliates,
	})))
}

func HandleAffiliateDashboard(kit *kit.Kit) error {
	idStr := chi.URLParam(kit.Request, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		kit.Response.WriteHeader(http.StatusBadRequest)
		return kit.Render(layouts.Base("Error", dashboard.ServersList(dashboard.ServersPageData{
			Error: "invalid affiliate ID",
		})))
	}

	affiliate, err := db.GetAffiliate(id)
	if err != nil {
		if kit.Request.Header.Get("HX-Request") == "true" {
			return kit.Render(dashboard.Index(dashboard.PageData{Error: err.Error()}))
		}
		return kit.Render(layouts.Base("Shop Dashboard", dashboard.Index(dashboard.PageData{
			Error: err.Error(),
		})))
	}

	apiKey, err := db.GenerateAndEnsureAPIKey(affiliate.ID)
	if err != nil {
		return kit.Render(layouts.Base("Shop Dashboard", dashboard.Index(dashboard.PageData{
			Error: err.Error(),
		})))
	}
	affiliate.APIKey = apiKey

	if affiliate.DashboardURL == "" {
		scheme := "http"
		if kit.Request.TLS != nil || kit.Request.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		autoURL := scheme + "://" + kit.Request.Host
		_ = db.UpdateAffiliateDashboardURL(affiliate.ID, autoURL)
		affiliate.DashboardURL = autoURL
	}

	client := services.NewShopClient(affiliate.ID)

	data := dashboard.PageData{
		Affiliate: affiliate,
	}

	orders, err := client.FetchOrders()
	if err != nil {
		data.Error = err.Error()
	} else {
		data.Orders = orders
	}

	comm, err := client.FetchCommission()
	if err != nil {
		if data.Error == "" {
			data.Error = err.Error()
		}
	} else {
		data.Commission = comm
	}

	errors, err := db.GetAffiliateErrors(affiliate.ID, 50)
	if err == nil {
		data.Errors = errors
	}

	if kit.Request.Header.Get("HX-Request") == "true" {
		return kit.Render(dashboard.Index(data))
	}
	return kit.Render(layouts.Base("Shop Dashboard - "+affiliate.Name, dashboard.Index(data)))
}

func HandleUpdateDomain(kit *kit.Kit) error {
	idStr := chi.URLParam(kit.Request, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		kit.Response.WriteHeader(http.StatusBadRequest)
		return kit.Render(dashboard.ShopURLDisplay(id, ""))
	}

	shopURL := kit.Request.FormValue("shop_url")
	if err := db.UpdateAffiliateShopURL(id, shopURL); err != nil {
		aff, fetchErr := db.GetAffiliate(id)
		if fetchErr != nil {
			return kit.Render(dashboard.ShopURLDisplay(id, ""))
		}
		return kit.Render(dashboard.ShopURLDisplay(id, aff.ShopURL))
	}

	return kit.Render(dashboard.ShopURLDisplay(id, shopURL))
}

func HandleUpdateDashboardURL(kit *kit.Kit) error {
	idStr := chi.URLParam(kit.Request, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		kit.Response.WriteHeader(http.StatusBadRequest)
		return kit.Render(dashboard.DashboardURLDisplay(id, ""))
	}

	url := kit.Request.FormValue("dashboard_url")
	if err := db.UpdateAffiliateDashboardURL(id, url); err != nil {
		aff, fetchErr := db.GetAffiliate(id)
		if fetchErr != nil {
			return kit.Render(dashboard.DashboardURLDisplay(id, ""))
		}
		return kit.Render(dashboard.DashboardURLDisplay(id, aff.DashboardURL))
	}

	return kit.Render(dashboard.DashboardURLDisplay(id, url))
}

func HandleReportError(kit *kit.Kit) error {
	apiKey := kit.Request.Header.Get("Authorization")
	if apiKey == "" || len(apiKey) < 8 {
		kit.Response.WriteHeader(http.StatusUnauthorized)
		return kit.Render(dashboard.ErrorResponse("missing or invalid authorization header"))
	}

	if len(apiKey) > 7 && apiKey[:7] == "Bearer " {
		apiKey = apiKey[7:]
	}

	aff, err := db.GetAffiliateByAPIKey(apiKey)
	if err != nil {
		kit.Response.WriteHeader(http.StatusUnauthorized)
		return kit.Render(dashboard.ErrorResponse("invalid api key"))
	}

	var body struct {
		Error       string `json:"error"`
		Path        string `json:"path"`
		Method      string `json:"method"`
		Host        string `json:"host"`
		Stack       string `json:"stack"`
		AffiliateID string `json:"affiliate_id"`
	}
	if err := json.NewDecoder(kit.Request.Body).Decode(&body); err != nil {
		kit.Response.WriteHeader(http.StatusBadRequest)
		return kit.Render(dashboard.ErrorResponse("invalid json body"))
	}

	if body.Error == "" {
		kit.Response.WriteHeader(http.StatusBadRequest)
		return kit.Render(dashboard.ErrorResponse("error field is required"))
	}

	errorType := body.Path
	if errorType == "" {
		errorType = "unknown"
	}
	details := fmt.Sprintf("method=%s host=%s", body.Method, body.Host)

	created, err := db.CreateAffiliateError(aff.ID, errorType, body.Error, details, body.Stack)
	if err != nil {
		kit.Response.WriteHeader(http.StatusInternalServerError)
		return kit.Render(dashboard.ErrorResponse("failed to store error"))
	}

	return kit.Render(dashboard.ErrorStored(created))
}
