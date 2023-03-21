package orders

import (
	"context"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/oblessing/artisgo/expert"
	"testing"
)

func Test_binanceAdapter_CloseTrade(t *testing.T) {
	type fields struct {
		client     *futures.Client
		isTestMode bool
	}
	type args struct {
		ctx    context.Context
		params expert.SellParams
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &binanceAdapter{
				client:     tt.fields.client,
				isTestMode: tt.fields.isTestMode,
			}
			got, err := b.CloseTrade(tt.args.ctx, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CloseTrade() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CloseTrade() got = %v, want %v", got, tt.want)
			}
		})
	}
}
