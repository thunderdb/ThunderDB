/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import (
	"time"

	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/merkle"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

//go:generate hsp

// Header is a block header.
type Header struct {
	Version     int32
	Producer    proto.NodeID
	GenesisHash hash.Hash
	ParentHash  hash.Hash
	MerkleRoot  hash.Hash
	Timestamp   time.Time
}

// SignedHeader is block header along with its producer signature.
type SignedHeader struct {
	Header
	HSV verifier.DefaultHashSignVerifierImpl
}

// Sign calls DefaultHashSignVerifierImpl to calculate header hash and sign it with signer.
func (s *SignedHeader) Sign(signer *ca.PrivateKey) error {
	return s.HSV.Sign(&s.Header, signer)
}

// Verify verifies the signature of the signed header.
func (s *SignedHeader) Verify() error {
	return s.HSV.Verify(&s.Header)
}

// VerifyAsGenesis verifies the signed header as a genesis block header.
func (s *SignedHeader) VerifyAsGenesis() (err error) {
	var pk *ca.PublicKey
	log.WithFields(log.Fields{
		"producer": s.Producer,
		"root":     s.GenesisHash.String(),
		"parent":   s.ParentHash.String(),
		"merkle":   s.MerkleRoot.String(),
		"block":    s.HSV.Hash().String(),
	}).Debug("Verifying genesis block header")
	if pk, err = kms.GetPublicKey(s.Producer); err != nil {
		return
	}
	if !pk.IsEqual(s.HSV.Signee) {
		err = ErrNodePublicKeyNotMatch
		return
	}
	return s.Verify()
}

// QueryAsTx defines a tx struct which is combined with request and signed response header
// for block.
type QueryAsTx struct {
	Request  *Request
	Response *SignedResponseHeader
}

// Block is a node of blockchain.
type Block struct {
	SignedHeader SignedHeader
	FailedReqs   []*Request
	QueryTxs     []*QueryAsTx
	Acks         []*Ack
}

// CalcNextID calculates the next query id by examinating every query in block, and adds write
// query number to the last offset.
//
// TODO(leventeliu): too tricky. Consider simply adding next id to each block header.
func (b *Block) CalcNextID() (id uint64, ok bool) {
	for _, v := range b.QueryTxs {
		if v.Request.Header.QueryType == WriteQuery {
			var nid = v.Response.LogOffset + uint64(len(v.Request.Payload.Queries))
			if nid > id {
				id = nid
			}
			ok = true
		}
	}
	return
}

// PackAndSignBlock generates the signature for the Block from the given PrivateKey.
func (b *Block) PackAndSignBlock(signer *ca.PrivateKey) (err error) {
	// Calculate merkle root
	b.SignedHeader.MerkleRoot = b.computeMerkleRoot()
	return b.SignedHeader.Sign(signer)
}

// Verify verifies the merkle root and header signature of the block.
func (b *Block) Verify() (err error) {
	// Verify merkle root
	if merkleRoot := b.computeMerkleRoot(); !merkleRoot.IsEqual(&b.SignedHeader.MerkleRoot) {
		return ErrMerkleRootVerification
	}
	return b.SignedHeader.Verify()
}

// VerifyAsGenesis verifies the block as a genesis block.
func (b *Block) VerifyAsGenesis() (err error) {
	var pk *ca.PublicKey
	if pk, err = kms.GetPublicKey(b.SignedHeader.Producer); err != nil {
		return
	}
	if !pk.IsEqual(b.SignedHeader.HSV.Signee) {
		err = ErrNodePublicKeyNotMatch
		return
	}
	return b.Verify()
}

// Timestamp returns the timestamp field of the block header.
func (b *Block) Timestamp() time.Time {
	return b.SignedHeader.Timestamp
}

// Producer returns the producer field of the block header.
func (b *Block) Producer() proto.NodeID {
	return b.SignedHeader.Producer
}

// ParentHash returns the parent hash field of the block header.
func (b *Block) ParentHash() *hash.Hash {
	return &b.SignedHeader.ParentHash
}

// BlockHash returns the parent hash field of the block header.
func (b *Block) BlockHash() *hash.Hash {
	return &b.SignedHeader.HSV.DataHash
}

// GenesisHash returns the parent hash field of the block header.
func (b *Block) GenesisHash() *hash.Hash {
	return &b.SignedHeader.GenesisHash
}

// Signee returns the signee field  of the block signed header.
func (b *Block) Signee() *ca.PublicKey {
	return b.SignedHeader.HSV.Signee
}

func (b *Block) computeMerkleRoot() hash.Hash {
	var hs = make([]*hash.Hash, 0, len(b.FailedReqs)+len(b.QueryTxs)+len(b.Acks))
	for i := range b.FailedReqs {
		h := b.FailedReqs[i].Header.Hash()
		hs = append(hs, &h)
	}
	for i := range b.QueryTxs {
		h := b.QueryTxs[i].Response.Hash()
		hs = append(hs, &h)
	}
	for i := range b.Acks {
		h := b.Acks[i].Header.Hash()
		hs = append(hs, &h)
	}
	return *merkle.NewMerkle(hs).GetRoot()
}

// Blocks is Block (reference) array.
type Blocks []*Block
