package handler

import (
	"context"
	"fmt"

	api "github.com/Neptune-Progate-Hackathon-AWS/back/internal/api"
)

// POST /navigation/route
func (s *Server) CalculateNavigationRoute(ctx context.Context, request api.CalculateNavigationRouteRequestObject) (api.CalculateNavigationRouteResponseObject, error) {
	result, err := s.navigationService.CalculateRoute(
		ctx,
		request.Body.Origin.Lat,
		request.Body.Origin.Lng,
		request.Body.Destination.Lat,
		request.Body.Destination.Lng,
		request.Body.DestName,
	)
	if err != nil {
		return nil, fmt.Errorf("navigation route calculation failed: %w", err)
	}

	// Build GeoJSON LineString coordinates
	coords := make([][]float64, len(result.Coordinates))
	copy(coords, result.Coordinates)

	return api.CalculateNavigationRoute200JSONResponse{
		DistanceMeters:  result.DistanceMeters,
		DurationSeconds: result.DurationSeconds,
		Polyline: api.GeoJSONLineString{
			Type:        api.LineString,
			Coordinates: coords,
		},
		SuggestionText: result.SuggestionText,
	}, nil
}
