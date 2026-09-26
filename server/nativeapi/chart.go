package nativeapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

func (api *Router) addChartRoute(r chi.Router) {
	r.Get("/chart", api.chartHandler)
}

func (api *Router) chartHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	opts, err := parseChartOptions(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if opts.Scope == model.ChartScopePersonal {
		user, ok := request.UserFrom(ctx)
		if !ok {
			http.Error(w, "authentication required for personal scope", http.StatusUnauthorized)
			return
		}
		opts.UserID = user.ID
	}

	entries, err := api.ds.Chart().GetChart(ctx, *opts)
	if err != nil {
		log.Error(ctx, "Error fetching scrobble history", "scope", opts.Scope, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := rest.RespondWithJSON(w, http.StatusOK, entries); err != nil {
		log.Error(ctx, "Error writing response", err)
	}
}

func parseChartOptions(q url.Values) (*model.ChartOptions, error) {
	limitStr := q.Get("limit")
	if limitStr == "" {
		return nil, fmt.Errorf("limit is required")
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return nil, fmt.Errorf("limit must be an integer")
	}
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	scope := model.ChartScope(q.Get("scope"))
	switch scope {
	case model.ChartScopeCommunity, model.ChartScopePersonal:
	default:
		return nil, fmt.Errorf("scope must be one of 'community', 'personal'")
	}

	yearStr := q.Get("year")
	if yearStr == "" {
		return nil, fmt.Errorf("year is required")
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1 {
		return nil, fmt.Errorf("year must be a valid positive integer")
	}

	monthStr := q.Get("month")
	weekStr := q.Get("week")

	switch {
	case monthStr != "" && weekStr != "":
		return nil, fmt.Errorf("specify only one of 'month' or 'week', not both")

	case monthStr != "":
		month, err := strconv.Atoi(monthStr)
		if err != nil || month < 1 || month > 12 {
			return nil, fmt.Errorf("month must be between 1 and 12")
		}
		return &model.ChartOptions{Limit: limit, Scope: scope, Year: year, Month: month}, nil

	case weekStr != "":
		week, err := strconv.Atoi(weekStr)
		if err != nil || week < 1 || week > 53 {
			return nil, fmt.Errorf("week must be between 1 and 53")
		}
		return &model.ChartOptions{Limit: limit, Scope: scope, Year: year, Week: week}, nil

	default:
		return &model.ChartOptions{Limit: limit, Scope: scope, Year: year}, nil
	}
}
