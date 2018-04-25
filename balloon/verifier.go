// Copyright © 2018 Banco Bilbao Vizcaya Argentaria S.A.  All rights reserved.
// Use of this source code is governed by an Apache 2 License
// that can be found in the LICENSE file

package balloon

import (
	"log"
	"os"
)

type Verifiable interface {
	Verify([]byte, []byte, uint) bool
}

type Proof struct {
	exists        bool
	hyperProof    Verifiable
	historyProof  Verifiable
	queryVersion  uint
	actualVersion uint
	log           *log.Logger
}

func NewProof(
	exists bool,
	hyperProof Verifiable,
	historyProof Verifiable,
	queryVersion uint,
	actualVersion uint,
) *Proof {
	return &Proof{
		exists,
		hyperProof,
		historyProof,
		queryVersion,
		actualVersion,
		log.New(os.Stdout, "BalloonProof", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile),
	}
}

func (p *Proof) Verify(commitment *Commitment, event []byte) bool {
	hyperCorrect := p.hyperProof.Verify(
		commitment.HyperDigest,
		event,
		p.queryVersion,
	)

	if p.exists {
		if p.queryVersion <= p.actualVersion {
			historyCorrect := p.historyProof.Verify(
				commitment.HistoryDigest,
				event,
				p.queryVersion,
			)
			return hyperCorrect && historyCorrect
		}
	}

	return hyperCorrect

}