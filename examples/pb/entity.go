package hello

import "github.com/vine-io/flow/api"

func (m *Echo) OwnerReferences() []*api.OwnerReference {
	return []*api.OwnerReference{}
}

func (m *Echo) Desc() string {
	return ""
}
