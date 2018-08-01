package modules

type KeypairManager struct {
	ResourceManager
}

var (
	Keypairs KeypairManager
)

func init() {
	Keypairs = KeypairManager{NewComputeManager("keypair", "keypairs",
		[]string{"ID", "Name", "Scheme", "Fingerprint", "Created_at", "Private_key_len", "Description", "Linked_guest_count"},
		[]string{})}

	registerCompute(&Keypairs)
}
