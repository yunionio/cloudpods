package tsdb

import (
	"math/rand"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type TSDBServiceSource struct {
	Type string
	URLs []string
}

func NewTSDBServiceSource(t string, urls []string) *TSDBServiceSource {
	return &TSDBServiceSource{
		Type: t,
		URLs: urls,
	}
}

func GetDefaultServiceSource(s *mcclient.ClientSession, endpointType string) (*TSDBServiceSource, error) {
	errs := []error{}
	for _, sType := range []string{apis.SERVICE_TYPE_INFLUXDB, apis.SERVICE_TYPE_VICTORIA_METRICS} {
		urls, err := s.GetServiceURLs(sType, endpointType)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "get %s service type %q", endpointType, sType))
		}
		if len(urls) != 0 {
			return NewTSDBServiceSource(sType, urls), nil
		}
	}
	return nil, errors.NewAggregate(errs)
}

func GetDefaultServiceSourceURLs(s *mcclient.ClientSession, endpointType string) ([]string, error) {
	src, err := GetDefaultServiceSource(s, endpointType)
	if err != nil {
		return nil, errors.Wrap(err, "GetDefaultServiceSource")
	}
	if len(src.URLs) == 0 {
		return nil, errors.Errorf("tsdb source %q URLs are empty", src.Type)
	}
	return src.URLs, nil
}

func GetDefaultServiceSourceURL(s *mcclient.ClientSession, endpointType string) (string, error) {
	urls, err := GetDefaultServiceSourceURLs(s, endpointType)
	if err != nil {
		return "", err
	}
	return urls[rand.Intn(len(urls))], nil
}
