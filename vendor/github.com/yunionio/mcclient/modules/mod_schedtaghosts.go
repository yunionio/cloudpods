package modules

var (
	Schedtaghosts JointResourceManager
)

func init() {
	Schedtaghosts = NewJointComputeManager("schedtaghost", "schedtaghosts",
		[]string{"Host_ID", "Host", "Schedtag_ID", "Schedtag"},
		[]string{},
		&Schedtags,
		&Hosts)
	registerCompute(&Schedtaghosts)
}
