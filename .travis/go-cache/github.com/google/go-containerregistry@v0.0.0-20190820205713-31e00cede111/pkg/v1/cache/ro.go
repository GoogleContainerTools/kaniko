package cache

import v1 "github.com/google/go-containerregistry/pkg/v1"

// ReadOnly returns a read-only implementation of the given Cache.
//
// Put and Delete operations are a no-op.
func ReadOnly(c Cache) Cache { return &ro{Cache: c} }

type ro struct{ Cache }

func (ro) Put(l v1.Layer) (v1.Layer, error) { return l, nil }
func (ro) Delete(v1.Hash) error             { return nil }
