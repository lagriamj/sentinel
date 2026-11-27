package scope

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestParseRejectsDisallowedTargets(t *testing.T) {
	tests := []struct {
		raw     string
		message string
	}{
		{raw: " ", message: "target url is required"},
		{raw: "http://127.0.0.1", message: "target host is not allowed"},
		{raw: "http://10.1.2.3/admin", message: "target host is not allowed"},
		{raw: "http://169.254.169.254/latest/meta-data", message: "target host is not allowed"},
		{raw: "http://localhost", message: "target host is not allowed"},
		{raw: "https://app.internal/health", message: "target host is not allowed"},
		{raw: "http://2130706433", message: "target host is not allowed"},
		{raw: "http://127.1", message: "target host is not allowed"},
		{raw: "https://user:secret@example.com", message: "target url must not include credentials"},
		{raw: "https://example.com/path?token=secret", message: "target url must not include a query or fragment"},
		{raw: "https://example.com/path#part", message: "target url must not include a query or fragment"},
		{raw: "ftp://example.com", message: "target url scheme must be http or https"},
	}
	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			_, _, err := Parse(test.raw)
			var rejected *Error
			if !errors.As(err, &rejected) || rejected.Message != test.message {
				t.Fatalf("error = %v, want %s", err, test.message)
			}
		})
	}
}

func TestParseNormalizesHost(t *testing.T) {
	target, hostIsIP, err := Parse("HTTPS://Example.COM:443/a/")
	if err != nil {
		t.Fatal(err)
	}
	if hostIsIP || target.URL != "https://example.com/a" || target.Host != "example.com" || target.Port != 443 {
		t.Fatalf("target = %+v ip = %v", target, hostIsIP)
	}
	literal, hostIsIP, err := Parse("https://203.0.113.10:8443/health")
	if err != nil || !hostIsIP || literal.Port != 8443 {
		t.Fatalf("literal = %+v ip = %v err = %v", literal, hostIsIP, err)
	}
}

type resolverStub struct {
	addresses []net.IPAddr
	err       error
	calls     int
}

func (r *resolverStub) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	return r.addresses, nil
}

func TestConfirmRejectsPrivateAnswer(t *testing.T) {
	resolver := &resolverStub{addresses: []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}, {IP: net.ParseIP("10.0.0.8")}}}
	err := Confirm(context.Background(), resolver, "rebind.example")
	var rejected *Error
	if !errors.As(err, &rejected) || rejected.Message != "target host is not allowed" {
		t.Fatalf("error = %v", err)
	}
}

func TestConfirmAllowsUnreachableResolver(t *testing.T) {
	resolver := &resolverStub{err: &net.DNSError{IsTemporary: true}}
	if err := Confirm(context.Background(), resolver, "example.com"); err != nil {
		t.Fatal(err)
	}
}

func TestConfirmRejectsMissingName(t *testing.T) {
	resolver := &resolverStub{err: &net.DNSError{IsNotFound: true}}
	err := Confirm(context.Background(), resolver, "missing.example")
	var rejected *Error
	if !errors.As(err, &rejected) || rejected.Message != "target host could not be resolved" {
		t.Fatalf("error = %v", err)
	}
}
