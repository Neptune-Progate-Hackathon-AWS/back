package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/location"
	loctypes "github.com/aws/aws-sdk-go-v2/service/location/types"
)

// NavigationService handles route calculation and suggestion generation.
type NavigationService struct {
	locationClient *location.Client
	bedrockClient  *bedrockruntime.Client
	calculatorName string
}

// RouteResult holds the result of a route calculation.
type RouteResult struct {
	DistanceMeters  float64
	DurationSeconds float64
	Coordinates     [][]float64 // [[lng, lat], ...]
	SuggestionText  string
}

// NewNavigationService creates a new NavigationService.
func NewNavigationService(locationClient *location.Client, bedrockClient *bedrockruntime.Client, calculatorName string) *NavigationService {
	return &NavigationService{
		locationClient: locationClient,
		bedrockClient:  bedrockClient,
		calculatorName: calculatorName,
	}
}

// CalculateRoute calculates a walking route from origin to destination using Amazon Location Service,
// and generates a Japanese suggestion text using Amazon Bedrock Claude 3 Haiku.
func (s *NavigationService) CalculateRoute(ctx context.Context, originLat, originLng, destLat, destLng float64, destName string) (*RouteResult, error) {
	// Call Location Service — CRITICAL: coordinates are [longitude, latitude] order
	out, err := s.locationClient.CalculateRoute(ctx, &location.CalculateRouteInput{
		CalculatorName:      aws.String(s.calculatorName),
		DeparturePosition:   []float64{originLng, originLat},
		DestinationPosition: []float64{destLng, destLat},
		TravelMode:          loctypes.TravelMode("Walking"),
		IncludeLegGeometry:  aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("calculate route: %w", err)
	}

	// Extract coordinates from Legs geometry
	var coords [][]float64
	for _, leg := range out.Legs {
		if leg.Geometry != nil {
			coords = append(coords, leg.Geometry.LineString...)
		}
	}

	// Summary.Distance is *float64 in km; convert to meters
	var distanceMeters float64
	if out.Summary.Distance != nil {
		distanceMeters = *out.Summary.Distance * 1000
	}

	var durationSeconds float64
	if out.Summary.DurationSeconds != nil {
		durationSeconds = *out.Summary.DurationSeconds
	}

	// Generate suggestion text (with fallback on any error)
	suggestionText := s.generateSuggestion(ctx, destName, distanceMeters, durationSeconds)

	return &RouteResult{
		DistanceMeters:  distanceMeters,
		DurationSeconds: durationSeconds,
		Coordinates:     coords,
		SuggestionText:  suggestionText,
	}, nil
}

// generateSuggestion calls Bedrock Claude 3 Haiku to generate a Japanese route guidance text.
// On any error, it returns a fallback text with distance and duration only.
func (s *NavigationService) generateSuggestion(ctx context.Context, destName string, distanceMeters, durationSeconds float64) string {
	fallback := fmt.Sprintf("%sまで徒歩約%.0f分（%.0fm）です。", destName, math.Round(durationSeconds/60), distanceMeters)

	prompt := fmt.Sprintf(
		"%sまで徒歩約%.0f分（%.0fm）の経路を、日本語で1文で自然に案内してください。",
		destName, math.Round(durationSeconds/60), distanceMeters,
	)

	reqBody, err := json.Marshal(map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        100,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": prompt},
				},
			},
		},
	})
	if err != nil {
		return fallback
	}

	resp, err := s.bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String("anthropic.claude-3-haiku-20240307-v1:0"),
		ContentType: aws.String("application/json"),
		Body:        reqBody,
	})
	if err != nil {
		return fallback
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil || len(result.Content) == 0 {
		return fallback
	}
	return result.Content[0].Text
}
