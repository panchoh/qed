package hyper2

import (
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/storage"
)

type PostOrderVisitor interface {
	VisitRoot(pos *Position, leftResult, rightResult interface{}) interface{}
	VisitNode(pos *Position, leftResult, rightResult interface{}) interface{}
	VisitPartialNode(pos *Position, leftResult interface{}) interface{}
	VisitLeaf(pos *Position, value []byte) interface{}
	VisitCached(pos *Position, cachedDigest hashing.Digest) interface{}
	VisitCollectable(pos *Position, result interface{}) interface{}
	VisitCacheable(pos *Position, result interface{}) interface{}
}

type ComputeHashVisitor struct {
	hasher hashing.Hasher
}

func NewComputeHashVisitor(hasher hashing.Hasher) *ComputeHashVisitor {
	return &ComputeHashVisitor{hasher}
}

func (v *ComputeHashVisitor) VisitRoot(pos *Position, leftResult, rightResult interface{}) interface{} {
	return v.hasher.Salted(pos.Bytes(), leftResult.(hashing.Digest), rightResult.(hashing.Digest))
}

func (v *ComputeHashVisitor) VisitNode(pos *Position, leftResult, rightResult interface{}) interface{} {
	return v.hasher.Salted(pos.Bytes(), leftResult.(hashing.Digest), rightResult.(hashing.Digest))
}

func (v *ComputeHashVisitor) VisitPartialNode(pos *Position, leftResult interface{}) interface{} {
	return v.hasher.Salted(pos.Bytes(), leftResult.(hashing.Digest))
}

func (v *ComputeHashVisitor) VisitLeaf(pos *Position, value []byte) interface{} {
	return v.hasher.Salted(pos.Bytes(), value)
}

func (v *ComputeHashVisitor) VisitCached(pos *Position, cachedDigest hashing.Digest) interface{} {
	return cachedDigest
}

func (v *ComputeHashVisitor) VisitCollectable(pos *Position, result interface{}) interface{} {
	return result
}

func (v *ComputeHashVisitor) VisitCacheable(pos *Position, result interface{}) interface{} {
	return result
}

type CachingVisitor struct {
	cache ModifiableCache

	*ComputeHashVisitor
}

func NewCachingVisitor(decorated *ComputeHashVisitor, cache ModifiableCache) *CachingVisitor {
	return &CachingVisitor{
		ComputeHashVisitor: decorated,
		cache:              cache,
	}
}

func (v *CachingVisitor) VisitCacheable(pos *Position, result interface{}) interface{} {
	v.cache.Put(pos, result.(hashing.Digest))
	return result
}

type CollectMutationsVisitor struct {
	storagePrefix byte
	mutations     []*storage.Mutation

	PostOrderVisitor
}

func NewCollectMutationsVisitor(decorated PostOrderVisitor, storagePrefix byte) *CollectMutationsVisitor {
	return &CollectMutationsVisitor{
		PostOrderVisitor: decorated,
		storagePrefix:    storagePrefix,
		mutations:        make([]*storage.Mutation, 0),
	}
}

func (v CollectMutationsVisitor) Result() []*storage.Mutation {
	return v.mutations
}

func (v *CollectMutationsVisitor) VisitCollectable(pos *Position, result interface{}) interface{} {
	value := v.PostOrderVisitor.VisitCollectable(pos, result).(hashing.Digest)
	v.mutations = append(v.mutations, storage.NewMutation(v.storagePrefix, pos.Bytes(), value))
	return result
}
