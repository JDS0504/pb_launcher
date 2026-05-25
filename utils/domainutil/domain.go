package domainutil

import (
	"fmt"
	"regexp"
	"strings"
)

func ToWildcardDomain(domain string) string {
	domain = strings.TrimPrefix(domain, "*.")
	return "*." + domain
}

func BaseDomain(domain string) string {
	if after, ok := strings.CutPrefix(domain, "*."); ok {
		return after
	}
	return domain
}

func IsWildcardDomain(domain string) bool {
	return strings.HasPrefix(domain, "*.")
}

func SubdomainMatchesWildcard(subdomain, wildcard string) bool {
	if !IsWildcardDomain(wildcard) {
		return false
	}
	wildcardBase := strings.TrimPrefix(wildcard, "*.")
	return strings.HasSuffix(subdomain, "."+wildcardBase) || subdomain == wildcardBase
}

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

func SanitizeToSlug(name string) string {
	slug := strings.ToLower(name)
	slug = slugRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func RootDomain(domainBase string) string {
	parts := strings.SplitN(domainBase, ".", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return domainBase
}

func GenerateFriendlyDomain(name string, domainBase string) (string, error) {
	slug := SanitizeToSlug(name)
	if slug == "" {
		return "", fmt.Errorf("invalid name")
	}
	return fmt.Sprintf("%s.%s", slug, RootDomain(domainBase)), nil
}


