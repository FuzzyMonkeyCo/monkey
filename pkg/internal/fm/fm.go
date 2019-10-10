package fm

type cltDoer interface {
	isSrv_Msg_Msg()
	do() (err error)
}

// Client TODO
type Client = FuzzyMonkey_DoClient

// func NewClient(ctx) (Client, func())
// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// // creds, err := credentials.NewClientTLSFromFile("ssl/mysrv.crt", "my.domain.tld")
// conn, err := grpc.DialContext(ctx, grpcHost,
// 	grpc.WithBlock(),
// 	grpc.WithInsecure(),
// 	// grpc.WithTransportCredentials(creds)),
// )
// clt := NewBifrostClient(conn)
// closer := func() {
// 	cancel()
// 	err = conn.Close()
// }
