package qemu

func init() {
	RegisterCmd(
		newCmd_10_0_7_x86_64(),
		newCmd_10_0_7_aarch64(),
		newCmd_10_0_7_riscv64(),
	)
}

type opt_1007_x86_64 struct {
	*baseOptions_x86_64
	*baseOptions_ge_310
	*baseOptions_ge_800_x86_64
}

func newOpt_10_0_7_x86_64() QemuOptions {
	return &opt_1007_x86_64{
		baseOptions_x86_64:        newBaseOptions_x86_64(),
		baseOptions_ge_800_x86_64: newBaseOptionsGE800_x86_64(),
		baseOptions_ge_310:        newBaseOptionsGE310(),
	}
}

func newCmd_10_0_7_x86_64() QemuCommand {
	return newBaseCommand(
		Version_10_0_7,
		Arch_x86_64,
		newOpt_10_0_7_x86_64(),
	)
}

type opt_1007_aarch64 struct {
	*baseOptions_aarch64
	*baseOptions_ge_310
}

func newCmd_10_0_7_aarch64() QemuCommand {
	return newBaseCommand(
		Version_10_0_7,
		Arch_aarch64,
		newOpt_10_0_7_aarch64(),
	)
}

func newOpt_10_0_7_aarch64() QemuOptions {
	return &opt_1007_aarch64{
		baseOptions_aarch64: newBaseOptions_aarch64(),
		baseOptions_ge_310:  newBaseOptionsGE310(),
	}
}

type opt_1007_riscv64 struct {
	*baseOptions_riscv64
	*baseOptions_ge_310
}

func newCmd_10_0_7_riscv64() QemuCommand {
	return newBaseCommand(
		Version_10_0_7,
		Arch_riscv64,
		newOpt_10_0_7_riscv64(),
	)
}

func newOpt_10_0_7_riscv64() QemuOptions {
	return &opt_1007_riscv64{
		baseOptions_riscv64: newBaseOptions_riscv64(),
		baseOptions_ge_310:  newBaseOptionsGE310(),
	}
}
