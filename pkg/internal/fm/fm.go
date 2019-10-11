package fm

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const grpcHost = "api.dev.fuzzymonkey.co"

type cltDoer interface {
	isSrv_Msg_Msg()
	do() (err error)
}

// Client is the gRPC clt-srv dialogue handler
type Client = FuzzyMonkey_DoClient

func NewClient(ctx context.Context, ua, apiKey string) (
	clt Client,
	closer func() error,
	err error,
) {
	// creds, err := credentials.NewClientTLSFromFile("ssl/mysrv.crt", "my.domain.tld")
	var conn *grpc.ClientConn
	if conn, err = grpc.DialContext(ctx, grpcHost,
		grpc.WithBlock(),
		grpc.WithInsecure(),
		// grpc.WithTransportCredentials(creds)),
	); err != nil {
		return
	}

	ctx = metadata.AppendToOutgoingContext(ctx,
		"ua", ua,
		"apiKey", apiKey,
	)

	if clt, err = NewFuzzyMonkeyClient(conn).Do(ctx); err != nil {
		return
	}
	closer = func() error { return conn.Close() }
	return
}
