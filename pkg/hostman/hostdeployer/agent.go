package hostdeployer

// func NewClient() {
// 	conn, err := grpc.Dial("localhost:1234", grpc.WithInsecure())
// 	defer conn.Close()
// 	if err != nil {
// 		//
// 	}

// 	client := deployapi.NewDeployAgentClient(conn)
// 	// client
// 	// r, err := client.DeployGuestFs(nil, nil)
// }

// func RunService() {
// 	grpcServer := grpc.NewServer()
// 	deployapi.RegisterDeployAgentServer(grpcServer, &DeployerServer{})
// 	listener, err := net.Listen("unix", "./deploy.sock")
// 	if err != nil {
// 		return err
// 	}
// 	grpcServer.Serve(listener)
// }
