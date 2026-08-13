package handlers

import (
	"net/http"
	"testing"

	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/aiproxy/chatlog"
)

func TestRecordMatchesFilterOwnerScope(t *testing.T) {
	own := chatlog.Record{ProjectID: "proj-a", DomainID: "dom-a"}
	otherProj := chatlog.Record{ProjectID: "proj-b", DomainID: "dom-a"}
	otherDom := chatlog.Record{ProjectID: "proj-c", DomainID: "dom-b"}
	empty := chatlog.Record{}

	projectFilter := usageFilter{Scope: rbacscope.ScopeProject, ProjectID: "proj-a", DomainID: "dom-a"}
	if !recordMatchesFilter(own, projectFilter) {
		t.Fatal("project scope should keep own project")
	}
	if recordMatchesFilter(otherProj, projectFilter) {
		t.Fatal("project scope should drop other project")
	}
	if recordMatchesFilter(empty, projectFilter) {
		t.Fatal("project scope should drop empty project_id")
	}

	domainFilter := usageFilter{Scope: rbacscope.ScopeDomain, ProjectID: "proj-a", DomainID: "dom-a"}
	if !recordMatchesFilter(own, domainFilter) || !recordMatchesFilter(otherProj, domainFilter) {
		t.Fatal("domain scope should keep same domain")
	}
	if recordMatchesFilter(otherDom, domainFilter) || recordMatchesFilter(empty, domainFilter) {
		t.Fatal("domain scope should drop other or empty domain")
	}

	systemFilter := usageFilter{Scope: rbacscope.ScopeSystem}
	for _, rec := range []chatlog.Record{own, otherProj, otherDom, empty} {
		if !recordMatchesFilter(rec, systemFilter) {
			t.Fatalf("system scope should keep %+v", rec)
		}
	}
}

func TestRecordMatchesFilterQueryProjectDomain(t *testing.T) {
	own := chatlog.Record{ProjectID: "proj-a", DomainID: "dom-a"}
	otherProj := chatlog.Record{ProjectID: "proj-b", DomainID: "dom-a"}
	otherDom := chatlog.Record{ProjectID: "proj-c", DomainID: "dom-b"}

	byProject := usageFilter{Scope: rbacscope.ScopeSystem, QueryProject: "proj-a"}
	if !recordMatchesFilter(own, byProject) {
		t.Fatal("query project should keep matching project")
	}
	if recordMatchesFilter(otherProj, byProject) || recordMatchesFilter(otherDom, byProject) {
		t.Fatal("query project should drop other projects")
	}

	byDomain := usageFilter{Scope: rbacscope.ScopeSystem, QueryDomain: "dom-a"}
	if !recordMatchesFilter(own, byDomain) || !recordMatchesFilter(otherProj, byDomain) {
		t.Fatal("query domain should keep same domain")
	}
	if recordMatchesFilter(otherDom, byDomain) {
		t.Fatal("query domain should drop other domain")
	}

	both := usageFilter{Scope: rbacscope.ScopeSystem, QueryProject: "proj-a", QueryDomain: "dom-a"}
	if !recordMatchesFilter(own, both) {
		t.Fatal("query project and domain should keep matching record")
	}
	if recordMatchesFilter(otherProj, both) || recordMatchesFilter(otherDom, both) {
		t.Fatal("query project and domain should drop mismatched records")
	}

	same := usageFilter{Scope: rbacscope.ScopeProject, ProjectID: "proj-a", DomainID: "dom-a", QueryProject: "proj-a"}
	if !recordMatchesFilter(own, same) {
		t.Fatal("project scope AND same query project should keep own record")
	}

	crossed := usageFilter{Scope: rbacscope.ScopeProject, ProjectID: "proj-a", DomainID: "dom-a", QueryProject: "proj-b"}
	if recordMatchesFilter(own, crossed) || recordMatchesFilter(otherProj, crossed) {
		t.Fatal("project scope AND other query project should match nothing")
	}
}

func TestParseUsageFilterProjectDomain(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/?project=proj-a&domain=dom-a&range=24h", nil)
	if err != nil {
		t.Fatal(err)
	}
	filter, err := parseUsageFilter(req)
	if err != nil {
		t.Fatal(err)
	}
	if filter.QueryProject != "proj-a" || filter.QueryDomain != "dom-a" {
		t.Fatalf("got project=%q domain=%q", filter.QueryProject, filter.QueryDomain)
	}

	req, err = http.NewRequest(http.MethodGet, "/?tenant_id=tid&project_domain=did&range=24h", nil)
	if err != nil {
		t.Fatal(err)
	}
	filter, err = parseUsageFilter(req)
	if err != nil {
		t.Fatal(err)
	}
	if filter.QueryProject != "tid" || filter.QueryDomain != "did" {
		t.Fatalf("alias got project=%q domain=%q", filter.QueryProject, filter.QueryDomain)
	}

	req, err = http.NewRequest(http.MethodGet, "/?project_id=pid&domain_id=did2&range=24h", nil)
	if err != nil {
		t.Fatal(err)
	}
	filter, err = parseUsageFilter(req)
	if err != nil {
		t.Fatal(err)
	}
	if filter.QueryProject != "pid" || filter.QueryDomain != "did2" {
		t.Fatalf("id alias got project=%q domain=%q", filter.QueryProject, filter.QueryDomain)
	}
}
