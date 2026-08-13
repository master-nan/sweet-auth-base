package service

import (
	"fmt"
	"strings"
)

type casbinPolicyStore interface {
	AddPolicy(params ...interface{}) (bool, error)
	RemovePolicy(params ...interface{}) (bool, error)
	GetFilteredPolicy(fieldIndex int, fieldValues ...string) ([][]string, error)
	ReplaceSubjectPolicies(subject string, policies [][]string) error
}

type casbinPolicyIdentity struct {
	Subject string
	Path    string
	Method  string
}

type casbinPolicySnapshot struct {
	casbinPolicyIdentity
	Existed bool
}

func quiesceCasbinPolicies(store casbinPolicyStore, identities []casbinPolicyIdentity) ([]casbinPolicySnapshot, error) {
	unique := make(map[casbinPolicyIdentity]struct{}, len(identities))
	snapshots := make([]casbinPolicySnapshot, 0, len(identities))
	for _, identity := range identities {
		identity.Subject = strings.TrimSpace(identity.Subject)
		identity.Path = strings.TrimSpace(identity.Path)
		identity.Method = strings.ToUpper(strings.TrimSpace(identity.Method))
		if identity.Subject == "" || identity.Path == "" || identity.Method == "" {
			continue
		}
		if _, ok := unique[identity]; ok {
			continue
		}
		unique[identity] = struct{}{}

		policies, err := store.GetFilteredPolicy(0, identity.Subject, identity.Path, identity.Method)
		if err != nil {
			_ = restoreCasbinPolicies(store, snapshots, nil)
			return nil, err
		}
		snapshot := casbinPolicySnapshot{casbinPolicyIdentity: identity, Existed: len(policies) > 0}
		snapshots = append(snapshots, snapshot)
		if snapshot.Existed {
			if _, err = store.RemovePolicy(identity.Subject, identity.Path, identity.Method); err != nil {
				_ = restoreCasbinPolicies(store, snapshots, nil)
				return nil, err
			}
		}
	}
	return snapshots, nil
}

func restoreCasbinPolicies(store casbinPolicyStore, snapshots []casbinPolicySnapshot, skip map[casbinPolicyIdentity]struct{}) error {
	for _, snapshot := range snapshots {
		if !snapshot.Existed {
			continue
		}
		if _, shouldSkip := skip[snapshot.casbinPolicyIdentity]; shouldSkip {
			continue
		}
		identity := snapshot.casbinPolicyIdentity
		if _, err := store.AddPolicy(identity.Subject, identity.Path, identity.Method); err != nil {
			return err
		}
	}
	return nil
}

func quiesceCasbinSubject(store casbinPolicyStore, subject string) ([][]string, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil, fmt.Errorf("casbin subject is required")
	}
	policies, err := store.GetFilteredPolicy(0, subject)
	if err != nil {
		return nil, err
	}
	if err = store.ReplaceSubjectPolicies(subject, nil); err != nil {
		return nil, err
	}
	return policies, nil
}
