package options

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type Route struct {
	Type        string
	Cidr        string
	NextHopType string
	NextHopId   string
}

type Routes []*Route

type RoutesOptions struct {
	RouteType        []string
	RouteCidr        []string
	RouteNextHopType []string
	RouteNextHopId   []string
}

func (opts *RoutesOptions) Params() (jsonutils.JSONObject, error) {
	len0 := len(opts.RouteType)
	len1 := len(opts.RouteCidr)
	if len0 != len1 || len0 != len(opts.RouteNextHopType) || len1 != len(opts.RouteNextHopId) {
		return nil, fmt.Errorf("there must be equal number of options of --route-xxx")
	}
	routes := []*Route{}
	for i := 0; i < len0; i++ {
		routes = append(routes, &Route{
			Type:        opts.RouteType[i],
			Cidr:        opts.RouteCidr[i],
			NextHopType: opts.RouteNextHopType[i],
			NextHopId:   opts.RouteNextHopId[i],
		})
	}
	routesJson := jsonutils.Marshal(routes)
	return routesJson, nil
}

type RouteTableCreateOptions struct {
	NAME string
	Vpc  string

	RoutesOptions
}

func (opts *RouteTableCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}
	routesJson, err := opts.RoutesOptions.Params()
	if err != nil {
		return nil, err
	}
	params.Set("routes", routesJson)
	return params, nil
}

type RouteTableGetOptions struct {
	ID string
}

type RouteTableUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	RoutesOptions
}

func (opts *RouteTableUpdateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}
	if len(opts.RouteCidr) != 0 {
		routesJson, err := opts.RoutesOptions.Params()
		if err != nil {
			return nil, err
		}
		params.Set("routes", routesJson)
	}
	return params, nil
}

type RouteTableAddRoutesOptions struct {
	ID string `json:"-"`

	RoutesOptions
}

func (opts *RouteTableAddRoutesOptions) Params() (*jsonutils.JSONDict, error) {
	if len(opts.RouteCidr) == 0 {
		return nil, fmt.Errorf("nothing to add")
	}
	routesJson, err := opts.RoutesOptions.Params()
	if err != nil {
		return nil, err
	}
	params := jsonutils.NewDict()
	params.Set("routes", routesJson)
	return params, nil
}

type RouteTableDelRoutesOptions struct {
	ID string `json:"-"`

	RouteCidr []string
}

func (opts *RouteTableDelRoutesOptions) Params() (*jsonutils.JSONDict, error) {
	if len(opts.RouteCidr) == 0 {
		return nil, fmt.Errorf("nothing to del")
	}
	params := jsonutils.NewDict()
	params.Set("cidrs", jsonutils.Marshal(opts.RouteCidr))
	return params, nil
}

type RouteTableDeleteOptions struct {
	ID string
}

type RouteTableListOptions struct {
	Manager     string
	Vpc         string
	Cloudregion string

	BaseListOptions
}
