package service

import (
	"errors"
	"reflect"
	"testing"
)

type memoryCasbinPolicyStore struct {
	policies map[casbinPolicyIdentity]bool
	addErr   error
}

func newMemoryCasbinPolicyStore(policies ...casbinPolicyIdentity) *memoryCasbinPolicyStore {
	store := &memoryCasbinPolicyStore{policies: make(map[casbinPolicyIdentity]bool, len(policies))}
	for _, policy := range policies {
		store.policies[policy] = true
	}
	return store
}

func (s *memoryCasbinPolicyStore) AddPolicy(params ...interface{}) (bool, error) {
	if s.addErr != nil {
		return false, s.addErr
	}
	policy := casbinPolicyIdentity{Subject: params[0].(string), Path: params[1].(string), Method: params[2].(string)}
	s.policies[policy] = true
	return true, nil
}

func (s *memoryCasbinPolicyStore) RemovePolicy(params ...interface{}) (bool, error) {
	policy := casbinPolicyIdentity{Subject: params[0].(string), Path: params[1].(string), Method: params[2].(string)}
	existed := s.policies[policy]
	delete(s.policies, policy)
	return existed, nil
}

func (s *memoryCasbinPolicyStore) GetFilteredPolicy(_ int, values ...string) ([][]string, error) {
	var result [][]string
	for policy := range s.policies {
		fields := []string{policy.Subject, policy.Path, policy.Method}
		matches := true
		for index, value := range values {
			if fields[index] != value {
				matches = false
			}
		}
		if matches {
			result = append(result, fields)
		}
	}
	return result, nil
}

func (s *memoryCasbinPolicyStore) ReplaceSubjectPolicies(subject string, policies [][]string) error {
	for policy := range s.policies {
		if policy.Subject == subject {
			delete(s.policies, policy)
		}
	}
	for _, policy := range policies {
		s.policies[casbinPolicyIdentity{Subject: policy[0], Path: policy[1], Method: policy[2]}] = true
	}
	return nil
}

func TestCasbinQuiesceRestoresOnlyPoliciesStillBackedByDatabase(t *testing.T) {
	first := casbinPolicyIdentity{Subject: "role", Path: "/first", Method: "GET"}
	second := casbinPolicyIdentity{Subject: "role", Path: "/second", Method: "POST"}
	store := newMemoryCasbinPolicyStore(first, second)

	snapshots, err := quiesceCasbinPolicies(store, []casbinPolicyIdentity{first, second})
	if err != nil || len(store.policies) != 0 {
		t.Fatalf("quiesce policies: snapshots=%v policies=%v err=%v", snapshots, store.policies, err)
	}
	if err := restoreCasbinPolicies(store, snapshots, map[casbinPolicyIdentity]struct{}{second: {}}); err != nil {
		t.Fatalf("restore policies: %v", err)
	}
	if !reflect.DeepEqual(store.policies, map[casbinPolicyIdentity]bool{first: true}) {
		t.Fatalf("unexpected restored policies: %#v", store.policies)
	}
}

func TestCasbinRestoreFailureRemainsFailClosed(t *testing.T) {
	policy := casbinPolicyIdentity{Subject: "role", Path: "/protected", Method: "GET"}
	store := newMemoryCasbinPolicyStore(policy)
	snapshots, err := quiesceCasbinPolicies(store, []casbinPolicyIdentity{policy})
	if err != nil {
		t.Fatalf("quiesce policies: %v", err)
	}
	store.addErr = errors.New("casbin unavailable")
	if err := restoreCasbinPolicies(store, snapshots, nil); err == nil {
		t.Fatal("expected restore failure")
	}
	if len(store.policies) != 0 {
		t.Fatalf("restore failure must not leave stale grant: %#v", store.policies)
	}
}
