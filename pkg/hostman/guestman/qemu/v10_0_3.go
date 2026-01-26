package qemu

func init() {
	RegisterCmd(
		newCmd_10_0_3_riscv64(),
	)
}

type opt_1003_riscv64 struct {
	*baseOptions_riscv64
	*baseOptions_ge_310
}

func newCmd_10_0_3_riscv64() QemuCommand {
	return newBaseCommand(
		Version_10_0_3,
		Arch_riscv64,
		newOpt_10_0_3_riscv64(),
	)
}

func newOpt_10_0_3_riscv64() QemuOptions {
	return &opt_1003_riscv64{
		baseOptions_riscv64: newBaseOptions_riscv64(),
		baseOptions_ge_310:  newBaseOptionsGE310(),
	}
}
