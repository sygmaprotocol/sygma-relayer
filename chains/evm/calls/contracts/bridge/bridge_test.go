package bridge

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts"
	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/types"
)

func TestBridgeContract_ProposalsHash(t *testing.T) {
	type fields struct {
		Contract contracts.Contract
	}
	type args struct {
		proposals []*proposal.Proposal
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		/*
			{
				name: "Test",
				fields: fields{
					Contract: contracts.Contract{},
				},
				args: args{
					proposals: []*proposal.Proposal{},
				},
			},
		*/
		{
			name: "Test",
			fields: fields{
				Contract: contracts.Contract{},
			},
			args: args{
				proposals: []*proposal.Proposal{
					{
						Source:       1,
						DepositNonce: 1,
						Data:         []byte{},
						ResourceId:   types.ResourceID{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &BridgeContract{
				Contract: tt.fields.Contract,
			}
			got, err := c.ProposalsHash(tt.args.proposals)
			if (err != nil) != tt.wantErr {
				t.Errorf("BridgeContract.ProposalsHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BridgeContract.ProposalsHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
