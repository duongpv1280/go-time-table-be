package auth

import (
	"context"
	_ "embed"
	"strings"

	domainAuth "gosample/internal/domain/auth"
)

//go:embed policies.csv
var policiesCSV string

type PermissionService struct {
	policies [][]string
}

func NewPermissionService() domainAuth.IPermissionService {
	return &PermissionService{policies: parsePolicies(policiesCSV)}
}

func (s *PermissionService) GetPermissionsForRole(_ context.Context, role string) (map[string]interface{}, error) {
	var rolePolices [][]string
	for _, p := range s.policies {
		if p[0] == role {
			rolePolices = append(rolePolices, p)
		}
	}
	return buildPermissions(rolePolices), nil
}

func parsePolicies(csv string) [][]string {
	var policies [][]string
	for _, line := range strings.Split(csv, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		if strings.TrimSpace(parts[0]) != "p" {
			continue
		}
		policies = append(policies, []string{
			strings.TrimSpace(parts[1]),
			strings.TrimSpace(parts[2]),
			strings.TrimSpace(parts[3]),
		})
	}
	return policies
}

// buildPermissions converts a role's Casbin policy rows into a nested permission map.
// Top-level resource permissions (e.g., /api/v1/teachers) subsume nested ones
// (e.g., /api/v1/teachers/:id/subjects → teachers.subjects is dropped if teachers is already set).
func buildPermissions(policies [][]string) map[string]interface{} {
	topLevel := map[string]string{}
	nested := map[string]map[string]string{}

	for _, p := range policies {
		if len(p) < 3 {
			continue
		}
		perm := methodToPerm(strings.TrimSpace(p[2]))
		segs := nonParamSegments(p[1])

		switch len(segs) {
		case 1:
			top := segs[0]
			if topLevel[top] != "write" {
				topLevel[top] = perm
			}
		case 2:
			top, sub := segs[0], segs[1]
			if nested[top] == nil {
				nested[top] = map[string]string{}
			}
			if nested[top][sub] != "write" {
				nested[top][sub] = perm
			}
		}
	}

	result := map[string]interface{}{}
	for res, perm := range topLevel {
		result[res] = perm
	}
	for res, subs := range nested {
		if _, exists := result[res]; exists {
			continue
		}
		inner := map[string]interface{}{}
		for sub, perm := range subs {
			inner[sub] = perm
		}
		result[res] = inner
	}
	return result
}

func methodToPerm(method string) string {
	if strings.ToUpper(method) == "GET" {
		return "read"
	}
	return "write"
}

func nonParamSegments(path string) []string {
	path = strings.TrimPrefix(strings.TrimSpace(path), "/api/v1/")
	parts := strings.Split(path, "/")
	var segs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && !strings.HasPrefix(p, ":") {
			segs = append(segs, p)
		}
	}
	return segs
}
