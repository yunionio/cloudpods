package modules

var Policies ResourceManager

func init() {
	Policies = NewIdentityV3Manager("policy", "policies",
		[]string{},
		[]string{})

	register(&Policies)
}
