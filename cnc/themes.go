package main

import (
	"net"
	"sort"
	"strings"
)

const (
	defaultThemeName  = "default"
	pinkCyanThemeName = "pink-cyan"

	bannersRelativeDir      = "banners/"
	bannersProjectFallback  = "cnc/banners/"
	defaultThemeHomeBanner  = "home.tfx"
	pinkCyanThemeHomeBanner = "home-pink-cyan.tfx"
	defaultThemeMethodsTFX  = "methods.tfx"
	pinkCyanMethodsTFX      = "methods-pink-cyan.tfx"
)

var (
	defaultThemeBannerPaths = []string{
		bannersRelativeDir + defaultThemeHomeBanner,
		bannersProjectFallback + defaultThemeHomeBanner,
	}
	pinkCyanThemeBannerPaths = []string{
		bannersRelativeDir + pinkCyanThemeHomeBanner,
		bannersProjectFallback + pinkCyanThemeHomeBanner,
	}
	defaultThemeMethodsBannerPaths = []string{
		bannersRelativeDir + defaultThemeMethodsTFX,
		bannersProjectFallback + defaultThemeMethodsTFX,
	}
	pinkCyanMethodsBannerPaths = []string{
		bannersRelativeDir + pinkCyanMethodsTFX,
		bannersProjectFallback + pinkCyanMethodsTFX,
	}
	adminThemeMethodsBannerPaths = map[string][]string{
		defaultThemeName:  defaultThemeMethodsBannerPaths,
		pinkCyanThemeName: pinkCyanMethodsBannerPaths,
	}
)

type adminThemeRenderer func(conn net.Conn, user *User)

var adminThemeRenderers = map[string]adminThemeRenderer{
	defaultThemeName:  writeDefaultAdminHeader,
	pinkCyanThemeName: writePinkCyanAdminHeader,
}

var adminThemeOrder = []string{
	defaultThemeName,
	pinkCyanThemeName,
}

func normalizeThemeName(theme string) string {
	return strings.ToLower(strings.TrimSpace(theme))
}

func availableAdminThemes() []string {
	seen := make(map[string]struct{}, len(adminThemeRenderers))
	names := make([]string, 0, len(adminThemeRenderers))

	for _, name := range adminThemeOrder {
		normalized := normalizeThemeName(name)
		if _, ok := adminThemeRenderers[normalized]; !ok {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		names = append(names, normalized)
	}

	extra := make([]string, 0)
	for name := range adminThemeRenderers {
		if _, ok := seen[name]; ok {
			continue
		}
		extra = append(extra, name)
	}
	sort.Strings(extra)

	return append(names, extra...)
}

func resolveThemeName(theme string) string {
	normalized := normalizeThemeName(theme)
	if normalized == "" {
		return defaultThemeName
	}
	if _, ok := adminThemeRenderers[normalized]; ok {
		return normalized
	}
	return defaultThemeName
}

func isKnownTheme(theme string) bool {
	_, ok := adminThemeRenderers[normalizeThemeName(theme)]
	return ok
}

func writeAdminHeader(conn net.Conn, user *User) {
	theme := defaultThemeName
	if user != nil {
		theme = resolveThemeName(user.Theme)
		user.Theme = theme
	}

	renderer, ok := adminThemeRenderers[theme]
	if !ok {
		renderer = adminThemeRenderers[defaultThemeName]
	}
	renderer(conn, user)
}

func writeThemeMethodsBanner(conn net.Conn, user *User) bool {
	theme := defaultThemeName
	if user != nil {
		theme = resolveThemeName(user.Theme)
		user.Theme = theme
	}

	paths, ok := adminThemeMethodsBannerPaths[theme]
	if ok && writeThemeBannerRawFromPaths(conn, paths) {
		return true
	}

	return writeThemeBannerRawFromPaths(conn, defaultThemeMethodsBannerPaths)
}
