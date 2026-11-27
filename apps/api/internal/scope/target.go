package scope

import (
	"context"
	"errors"
	"net"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const resolveTimeout = 2 * time.Second

// Error is a target the policy rejects. The message is safe to show to an operator.
type Error struct {
	Message string
}

func (e *Error) Error() string {
	if e == nil || e.Message == "" {
		return "target url is invalid"
	}
	return e.Message
}

// Target is a normalized http(s) origin and optional path.
type Target struct {
	URL    string
	Scheme string
	Host   string
	Port   int
}

// Resolver looks up the addresses of a hostname.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

const maxTargetURLLength = 2048

// Parse normalizes an operator-supplied target URL.
// The boolean is true when the host is a literal IP address.
func Parse(raw string) (Target, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Target{}, false, invalid("target url is required")
	}
	if len(raw) > maxTargetURLLength || strings.Contains(raw, "\\") {
		return Target{}, false, invalid("target url is invalid")
	}
	for _, r := range raw {
		if unicode.IsControl(r) || r > unicode.MaxASCII {
			return Target{}, false, invalid("target url is invalid")
		}
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.Opaque != "" {
		return Target{}, false, invalid("target url is invalid")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return Target{}, false, invalid("target url scheme must be http or https")
	}
	if parsed.User != nil {
		return Target{}, false, invalid("target url must not include credentials")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return Target{}, false, invalid("target url must not include a query or fragment")
	}

	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" {
		return Target{}, false, invalid("target url is invalid")
	}
	port, err := targetPort(scheme, parsed.Port())
	if err != nil {
		return Target{}, false, err
	}
	urlPath, err := targetPath(parsed.Path)
	if err != nil {
		return Target{}, false, err
	}
	if disallowedHostname(host) || ambiguousNumericHost(host) {
		return Target{}, false, invalid("target host is not allowed")
	}

	target := Target{
		URL:    formatTargetURL(scheme, host, port, urlPath),
		Scheme: scheme,
		Host:   host,
		Port:   port,
	}
	if ip := net.ParseIP(host); ip != nil {
		if !publicIP(ip) {
			return Target{}, false, invalid("target host is not allowed")
		}
		return target, true, nil
	}
	if !validHost(host) {
		return Target{}, false, invalid("target url is invalid")
	}
	return target, false, nil
}

func targetPort(scheme, raw string) (int, error) {
	if raw == "" {
		if scheme == "https" {
			return 443, nil
		}
		return 80, nil
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return 0, invalid("target url is invalid")
	}
	return port, nil
}

func targetPath(raw string) (string, error) {
	if raw == "" || raw == "/" {
		return "", nil
	}
	if !strings.HasPrefix(raw, "/") || strings.Contains(raw, "..") {
		return "", invalid("target url is invalid")
	}
	cleaned := path.Clean(raw)
	if cleaned == "/" || cleaned == "." {
		return "", nil
	}
	if !strings.HasPrefix(cleaned, "/") || strings.Contains(cleaned, "..") {
		return "", invalid("target url is invalid")
	}
	return cleaned, nil
}

func formatTargetURL(scheme, host string, port int, urlPath string) string {
	hostPort := host
	defaultPort := (scheme == "https" && port == 443) || (scheme == "http" && port == 80)
	if !defaultPort {
		hostPort = net.JoinHostPort(host, strconv.Itoa(port))
	} else if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		hostPort = "[" + host + "]"
	}
	return (&url.URL{Scheme: scheme, Host: hostPort, Path: urlPath}).String()
}

func validHost(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.To16() != nil
	}
	if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") || strings.Contains(host, "..") {
		return false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return false
			}
		}
	}
	return true
}

func ambiguousNumericHost(host string) bool {
	if net.ParseIP(host) != nil {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if !numericLabel(label) {
			return false
		}
	}
	return len(labels) > 0
}

func numericLabel(label string) bool {
	if label == "" {
		return false
	}
	if len(label) > 2 && (label[0] == '0' && (label[1] == 'x' || label[1] == 'X')) {
		_, err := strconv.ParseUint(label[2:], 16, 64)
		return err == nil
	}
	_, err := strconv.ParseUint(label, 10, 32)
	return err == nil
}

func disallowedHostname(host string) bool {
	switch host {
	case "localhost", "metadata", "metadata.google.internal", "metadata.goog":
		return true
	}
	return strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".internal")
}

func publicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	return true
}

// Confirm checks a hostname against the resolver.
// A missing name is rejected. A resolver that cannot be reached is allowed,
// because the API network often has no route to public DNS. Execution must
// resolve the name again before any request.
func Confirm(ctx context.Context, resolver Resolver, host string) error {
	if resolver == nil {
		return errors.New("target resolver is unavailable")
	}
	lookupCtx, cancel := context.WithTimeout(ctx, resolveTimeout)
	defer cancel()
	addresses, err := resolver.LookupIPAddr(lookupCtx, host)
	if err != nil || len(addresses) == 0 {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			return invalid("target host could not be resolved")
		}
		return nil
	}
	for _, address := range addresses {
		if !publicIP(address.IP) {
			return invalid("target host is not allowed")
		}
	}
	return nil
}

func invalid(message string) error {
	return &Error{Message: message}
}
