package provider

import "testing"

func TestParseDomainListForSingleVMEnvelope(t *testing.T) {
	domains, err := parseDomainList([]byte(`{"domains":[{"domain":"gateway.example.com"}],"vm_name":"router"}`))
	if err != nil {
		t.Fatalf("parseDomainList() error = %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("len(domains) = %d, want 1", len(domains))
	}
	if domains[0].VMName != "router" || domains[0].Domain != "gateway.example.com" {
		t.Fatalf("domain = %+v", domains[0])
	}
}

func TestParseDomainListForAllDomainsEnvelope(t *testing.T) {
	domains, err := parseDomainList([]byte(`{"domains":[{"vm_name":"router","domain":"gateway.example.com"}]}`))
	if err != nil {
		t.Fatalf("parseDomainList() error = %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("len(domains) = %d, want 1", len(domains))
	}
	if domains[0].VMName != "router" || domains[0].Domain != "gateway.example.com" {
		t.Fatalf("domain = %+v", domains[0])
	}
}

func TestSplitDomainResourceID(t *testing.T) {
	vmName, domain, err := splitDomainResourceID("router/gateway.example.com")
	if err != nil {
		t.Fatalf("splitDomainResourceID() error = %v", err)
	}
	if vmName != "router" || domain != "gateway.example.com" {
		t.Fatalf("splitDomainResourceID() = %q, %q", vmName, domain)
	}
}

func TestSplitDomainResourceIDRejectsInvalidID(t *testing.T) {
	if _, _, err := splitDomainResourceID("gateway.example.com"); err == nil {
		t.Fatal("splitDomainResourceID() error = nil, want error")
	}
}
