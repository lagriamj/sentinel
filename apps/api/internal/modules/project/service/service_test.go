package service

import (
	"context"
	"errors"
	"net"
	"testing"

	"sentinel-scanner/apps/api/internal/modules/project/model"
	"sentinel-scanner/apps/api/internal/scope"
)

const (
	ownerID  = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	memberID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
)

type memStore struct {
	projects map[string]model.Project
	members  map[string][]model.Member
	targets  []model.Target
	users    map[string]string
	next     int
}

func newMem() *memStore {
	return &memStore{
		projects: map[string]model.Project{},
		members:  map[string][]model.Member{},
		users:    map[string]string{ownerID: "owner@example.com", memberID: "member@example.com"},
	}
}

func (m *memStore) id() string {
	m.next++
	return "11111111-1111-4111-8111-11111111111" + string(rune('0'+m.next))
}

func (m *memStore) role(userID, projectID string) (string, bool) {
	for _, member := range m.members[projectID] {
		if member.UserID == userID {
			return member.Role, true
		}
	}
	return "", false
}

func (m *memStore) CreateProject(_ context.Context, userID, name, description string) (model.Project, error) {
	project := model.Project{ID: m.id(), Name: name, Description: description, Role: model.RoleOwner}
	m.projects[project.ID] = project
	m.members[project.ID] = []model.Member{{UserID: userID, Email: m.users[userID], Role: model.RoleOwner}}
	return project, nil
}

func (m *memStore) ListProjects(context.Context, string, int, int) (model.ProjectPage, error) {
	return model.ProjectPage{}, nil
}

func (m *memStore) GetProject(_ context.Context, userID, projectID string) (model.Project, error) {
	project, ok := m.projects[projectID]
	if !ok {
		return model.Project{}, model.ErrNotFound
	}
	role, member := m.role(userID, projectID)
	if !member {
		return model.Project{}, model.ErrNotFound
	}
	project.Role = role
	return project, nil
}

func (m *memStore) UpdateProject(_ context.Context, userID, projectID, name, description string) (model.Project, error) {
	role, member := m.role(userID, projectID)
	if !member {
		return model.Project{}, model.ErrNotFound
	}
	if role != model.RoleOwner {
		return model.Project{}, model.ErrForbidden
	}
	project := m.projects[projectID]
	project.Name = name
	project.Description = description
	project.Role = role
	m.projects[projectID] = project
	return project, nil
}

func (m *memStore) DeleteProject(_ context.Context, userID, projectID string) error {
	role, member := m.role(userID, projectID)
	if !member {
		return model.ErrNotFound
	}
	if role != model.RoleOwner {
		return model.ErrForbidden
	}
	for _, target := range m.targets {
		if target.ProjectID == projectID {
			return model.ErrHasTargets
		}
	}
	delete(m.projects, projectID)
	return nil
}

func (m *memStore) ListMembers(context.Context, string, string) ([]model.Member, error) {
	return nil, nil
}

func (m *memStore) AddMember(_ context.Context, actorID, projectID, email, role string) (model.Member, error) {
	actor, member := m.role(actorID, projectID)
	if !member {
		return model.Member{}, model.ErrNotFound
	}
	if actor != model.RoleOwner {
		return model.Member{}, model.ErrForbidden
	}
	var userID string
	for id, stored := range m.users {
		if stored == email {
			userID = id
		}
	}
	if userID == "" {
		return model.Member{}, model.ErrUserNotFound
	}
	if _, exists := m.role(userID, projectID); exists {
		return model.Member{}, model.ErrDuplicateMember
	}
	added := model.Member{UserID: userID, Email: email, Role: role}
	m.members[projectID] = append(m.members[projectID], added)
	return added, nil
}

func (m *memStore) RemoveMember(_ context.Context, actorID, projectID, memberUserID string) error {
	actor, member := m.role(actorID, projectID)
	if !member {
		return model.ErrNotFound
	}
	if actor != model.RoleOwner {
		return model.ErrForbidden
	}
	role, exists := m.role(memberUserID, projectID)
	if !exists {
		return model.ErrMemberNotFound
	}
	if role == model.RoleOwner {
		owners := 0
		for _, item := range m.members[projectID] {
			if item.Role == model.RoleOwner {
				owners++
			}
		}
		if owners < 2 {
			return model.ErrLastOwner
		}
	}
	next := m.members[projectID][:0]
	for _, item := range m.members[projectID] {
		if item.UserID != memberUserID {
			next = append(next, item)
		}
	}
	m.members[projectID] = next
	return nil
}

func (m *memStore) ListTargets(context.Context, string, string, int, int) (model.TargetPage, error) {
	return model.TargetPage{}, nil
}

func (m *memStore) RegisterTarget(_ context.Context, userID, projectID string, target scope.Target) (model.Target, error) {
	if _, member := m.role(userID, projectID); !member {
		return model.Target{}, model.ErrNotFound
	}
	for _, existing := range m.targets {
		if existing.ProjectID == projectID && existing.URL == target.URL {
			return model.Target{}, model.ErrDuplicateTarget
		}
	}
	stored := model.Target{
		ID: m.id(), ProjectID: projectID, URL: target.URL, Scheme: target.Scheme,
		Host: target.Host, Port: target.Port, Authorized: true,
	}
	m.targets = append(m.targets, stored)
	return stored, nil
}

func (m *memStore) DeleteTarget(_ context.Context, userID, projectID, targetID string) error {
	role, member := m.role(userID, projectID)
	if !member {
		return model.ErrNotFound
	}
	if role != model.RoleOwner {
		return model.ErrForbidden
	}
	for i, target := range m.targets {
		if target.ID == targetID && target.ProjectID == projectID {
			m.targets = append(m.targets[:i], m.targets[i+1:]...)
			return nil
		}
	}
	return model.ErrTargetNotFound
}

func (m *memStore) FindForMember(_ context.Context, userID, targetID string) (model.Target, error) {
	for _, target := range m.targets {
		if target.ID != targetID {
			continue
		}
		if _, member := m.role(userID, target.ProjectID); !member {
			return model.Target{}, model.ErrTargetNotFound
		}
		return target, nil
	}
	return model.Target{}, model.ErrTargetNotFound
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

func TestProjectMembershipAndTargets(t *testing.T) {
	store := newMem()
	resolver := &resolverStub{addresses: []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}}}
	svc := New(store, resolver)
	ctx := context.Background()

	project, err := svc.CreateProject(ctx, ownerID, "  Web  ", "Primary")
	if err != nil || project.Role != model.RoleOwner || project.Name != "Web" {
		t.Fatalf("project = %+v err = %v", project, err)
	}
	if _, err := svc.GetProject(ctx, memberID, project.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("outsider get = %v", err)
	}
	authorized := true
	denied := false
	if _, err := svc.RegisterTarget(ctx, ownerID, project.ID, "http://127.0.0.1", &authorized); err == nil {
		t.Fatal("loopback target was stored")
	}
	if _, err := svc.RegisterTarget(ctx, ownerID, project.ID, "https://example.com", &denied); err == nil {
		t.Fatal("unauthorized target was stored")
	}
	target, err := svc.RegisterTarget(ctx, ownerID, project.ID, "HTTPS://Example.COM/a/", &authorized)
	if err != nil || target.URL != "https://example.com/a" || resolver.calls != 1 {
		t.Fatalf("target = %+v err = %v calls = %d", target, err, resolver.calls)
	}
	if _, err := svc.RegisterTarget(ctx, ownerID, project.ID, "https://example.com/a", &authorized); err == nil {
		t.Fatal("duplicate target was stored")
	}
	if err := svc.DeleteProject(ctx, ownerID, project.ID); err == nil {
		t.Fatal("project with a target was deleted")
	}
	if err := svc.RemoveMember(ctx, ownerID, project.ID, ownerID); err == nil {
		t.Fatal("last owner was removed")
	}
	member, err := svc.AddMember(ctx, ownerID, project.ID, "member@example.com", model.RoleMember)
	if err != nil || member.UserID != memberID {
		t.Fatalf("member = %+v err = %v", member, err)
	}
	if _, err := svc.UpdateProject(ctx, memberID, project.ID, "Changed", ""); !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("member update = %v", err)
	}
	found, err := svc.FindForMember(ctx, memberID, target.ID)
	if err != nil || found.ID != target.ID {
		t.Fatalf("member target = %+v err = %v", found, err)
	}
}
