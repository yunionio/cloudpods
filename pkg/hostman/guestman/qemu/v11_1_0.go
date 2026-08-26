package qemu

func init() {
	RegisterCmd(newCmd_11_1_0_riscv64())
}

type opt_1110_riscv64 struct {
	*baseOptions_riscv64
	*baseOptions_ge_310
}

func newCmd_11_1_0_riscv64() QemuCommand {
	return newBaseCommand(
		Version_11_1_0,
		Arch_riscv64,
		newOpt_11_1_0_riscv64(),
	)
}

func newOpt_11_1_0_riscv64() QemuOptions {
	return &opt_1110_riscv64{
		baseOptions_riscv64: newBaseOptions_riscv64(),
		baseOptions_ge_310:  newBaseOptionsGE310(),
	}
}
