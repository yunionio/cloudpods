package victoriametrics

func newTestClient(ep string) (Client, error) {
	return NewClient(ep)
}

/*func Test_client_QueryRange(t *testing.T) {
	cli, err := newTestClient("http://192.168.222.171:34795/")
	if err != nil {
		t.Fatalf("newTestClient")
	}
	resp, err := cli.QueryRange(context.Background(), http.DefaultClient, `avg by(host_id) (avg_over_time(cpu_usage_active{res_type="host"}))`, time.Second*50, &TimeRange{
		Start: 1698724482.213,
		End:   1698746082.213,
	}, true)
	if err != nil {
		t.Fatalf("query err: %v", err)
	}
	log.Infof("get resp: %s", jsonutils.Marshal(resp).PrettyString())
}*/
