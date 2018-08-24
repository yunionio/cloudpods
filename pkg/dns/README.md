# 行为约定

A

```sh
names=''
names="$names titan" #ok, guest PlainName
names="$names titan.hq.cloud.yunionyun.com" #ok, guest CloudZoneFQDN
names="$names kubenode" #ok, host PlainName
names="$names kubenode.hq.cloud.yunionyun.com" #ok, host CloudZoneFQDN
names="$names whoever-the-ether" #NXDOMAIN, nonexistent PlainName
names="$names whoever-the-ether.titan.hq.cloud.yunionyun.com" #NXDOMAIN, nonexistent CloudZoneFQDN
names="$names mail.google.com" #ok, dnsrecords
names="$names www.douban.com" #ok, pub
names="$names app" #ok, k8s svc PlainName in "default" ns
names="$names app.default" #NXDOMAIN, k8s svc name.namespace
names="$names app.default.hq.cloud.yunionyun.com" #ok, k8s name.ns CloudZoneFQDN
names="$names mon-kafka.system" #NXDOMAIN, k8s svc name.namespace
names="$names mon-kafka.system.hq.cloud.yunionyun.com" #ok, k8s name.ns CloudZoneFQDN

# TODO source IP
# TODO other TYPEs
for name in $names; do
	echo "############### $name"
	#dig @192.168.222.171 $name
	#dig -p 54 @10.168.222.136 $name
	dig -p 54 @192.168.222.171 $name
done
```

PTR

```sh
names=''
names="$names " #ok, guest ip
names="$names " #ok, host ip
names="$names " #ok, ptr records in db
names="$names " #NXDOMAIN, others

for name in $names; do
	echo "############### $name"
	dig -p 54 @192.168.222.171 -x $name
done
```

# 配置

	log {
		# note that apart from rcode like NXDOMAIN, SERVFAIL, coredns will also
		# log NOERROR response when it's NoData as defined by coredns itself
		#
		# > NoData indicates name found, but not the type: NOERROR in header, SOA in auth.
		#
		class denial
		class error
	}
