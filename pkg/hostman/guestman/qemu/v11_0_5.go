package qemu

func init() {
	RegisterCmd(
		newCmd_11_0_5_x86_64(),
		newCmd_11_0_5_aarch64(),
		newCmd_11_0_5_riscv64(),
	)
}

type opt_1105_x86_64 struct {
	*opt_1007_x86_64
	*baseOptions_ge_1105
}

func newOpt_11_0_5_x86_64() QemuOptions {
	return &opt_1105_x86_64{
		opt_1007_x86_64: newOpt_10_0_7_x86_64().(*opt_1007_x86_64),
	}
}

func newCmd_11_0_5_x86_64() QemuCommand {
	return newBaseCommand(
		Version_11_0_5,
		Arch_x86_64,
		newOpt_11_0_5_x86_64(),
	)
}

type opt_1105_aarch64 struct {
	*opt_1007_aarch64
	*baseOptions_ge_1105
}

func newCmd_11_0_5_aarch64() QemuCommand {
	return newBaseCommand(
		Version_11_0_5,
		Arch_aarch64,
		newOpt_11_0_5_aarch64(),
	)
}

func newOpt_11_0_5_aarch64() QemuOptions {
	return &opt_1105_aarch64{
		opt_1007_aarch64: newOpt_10_0_7_aarch64().(*opt_1007_aarch64),
	}
}

type opt_1105_riscv64 struct {
	*opt_1007_riscv64
	*baseOptions_ge_1105
}

func newCmd_11_0_5_riscv64() QemuCommand {
	return newBaseCommand(
		Version_11_0_5,
		Arch_riscv64,
		newOpt_11_0_5_riscv64(),
	)
}

func newOpt_11_0_5_riscv64() QemuOptions {
	return &opt_1105_riscv64{
		opt_1007_riscv64: newOpt_10_0_7_riscv64().(*opt_1007_riscv64),
	}
}
