package forwarder

import (
	"google.golang.org/grpc"

	"yunion.io/x/pkg/errors"

	pb "yunion.io/x/onecloud/pkg/hostman/guestman/forwarder/api"
)

func NewClient(target string) (pb.ForwarderClient, error) {
	cc, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrap(err, "grpc Dial")
	}
	c := pb.NewForwarderClient(cc)
	return c, nil
}
